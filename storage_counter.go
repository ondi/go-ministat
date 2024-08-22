//
//
//

package ministat

import (
	"sync"
	"time"

	"github.com/ondi/go-cache"
	"github.com/ondi/go-unique"
)

type Counter_t struct {
	median          *Median_t[time.Duration]
	average         *Average_t[time.Duration] // RPS
	hit_begin_ts    time.Time
	hit_end_median  time.Duration
	hit_end_max     time.Duration
	hit_end_average time.Duration
	hit_end_size    int
	rps             int
	sampling        int64
	hits            int64
	pending         int64
	processed       int64
	errors          int64
}

type Result_t struct {
	Hits       int64
	Pending    int64
	Processed  int64
	Errors     int64
	HitBeginTs time.Time
	Last       [4]Duration_t
	Current    [4]Duration_t
}

func (self *Counter_t) CounterAdd(a int64) {
	self.sampling += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.sampling
}

func NoEvict[Key_t comparable](page Key_t, value *Counter_t) {}

type Storage_t[Key_t comparable] struct {
	mx           sync.Mutex
	pages        *unique.Often_t[Key_t, *Counter_t]
	median_ttl   time.Duration
	median_limit int
}

func NewStorage[Key_t comparable](limit_pages int, median_limit int, median_ttl time.Duration, evict func(page Key_t, value *Counter_t)) (self *Storage_t[Key_t]) {
	self = &Storage_t[Key_t]{
		pages:        unique.NewOften(limit_pages, evict),
		median_ttl:   median_ttl,
		median_limit: median_limit,
	}
	return
}

func (self *Storage_t[Key_t]) HitBegin(name Key_t, begin time.Time) (counter *Counter_t, sampling int64, pending int64, dur [1]Duration_t) {
	self.mx.Lock()
	counter, _ = self.pages.Add(
		name,
		func(p **Counter_t) {
			*p = &Counter_t{
				median:  NewMedian[time.Duration](self.median_limit, self.median_ttl),
				average: NewAverage[time.Duration](32, 1*time.Second),
			}
		},
	)
	counter.hits++
	counter.pending++
	_, counter.rps = counter.average.Add(begin, 0)
	counter.hit_begin_ts = begin
	sampling = counter.sampling
	pending = counter.pending
	dur[0].Label = "rps"
	dur[0].Size = counter.rps
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitEnd(counter *Counter_t, name Key_t, begin time.Time, end time.Time, processed int64, errors int64) (out [3]Duration_t) {
	self.mx.Lock()
	counter.pending--
	counter.errors += errors
	counter.processed += processed
	diff := end.Sub(begin)
	counter.hit_end_median, counter.hit_end_max, counter.hit_end_average, counter.hit_end_size = counter.median.Add(end, diff)
	out[0].Label = "med"
	out[1].Label = "max"
	out[2].Label = "avg"
	out[0].Duration = counter.hit_end_median
	out[1].Duration = counter.hit_end_max
	out[2].Duration = counter.hit_end_average
	out[0].Size = counter.hit_end_size
	out[1].Size = counter.hit_end_size
	out[2].Size = counter.hit_end_size
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitGet(name Key_t, ts time.Time) (out Result_t, ok bool) {
	self.mx.Lock()
	res, ok := self.pages.Get(name)
	if ok {
		out = ToResult(res, ts)
	}
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) RangeSort(order cache.Less_t[Key_t, *Counter_t], ts time.Time, f func(name Key_t, res Result_t) bool) {
	self.mx.Lock()
	self.pages.RangeSort(
		order,
		func(key Key_t, value *Counter_t) bool {
			return f(key, ToResult(value, ts))
		},
	)
	self.mx.Unlock()
}

func (self *Storage_t[Key_t]) Range(ts time.Time, f func(name Key_t, res Result_t) bool) {
	self.mx.Lock()
	self.pages.Range(
		func(key Key_t, value *Counter_t) bool {
			return f(key, ToResult(value, ts))
		},
	)
	self.mx.Unlock()
}

type Less_t[Key_t comparable] struct {
	cache.Less_t[Key_t, *Counter_t]
}

func LessHits[Key_t comparable](a *cache.Value_t[Key_t, *Counter_t], b *cache.Value_t[Key_t, *Counter_t]) bool {
	return a.Value.hits < b.Value.hits
}

func LessProcessed[Key_t comparable](a *cache.Value_t[Key_t, *Counter_t], b *cache.Value_t[Key_t, *Counter_t]) bool {
	return a.Value.processed < b.Value.processed
}

func LessDuration[Key_t comparable](a *cache.Value_t[Key_t, *Counter_t], b *cache.Value_t[Key_t, *Counter_t]) bool {
	return a.Value.median.median.Value.Data < b.Value.median.median.Value.Data
}

func ToResult(in *Counter_t, ts time.Time) (out Result_t) {
	out.Hits = in.hits
	out.Pending = in.pending
	out.Processed = in.processed
	out.Errors = in.errors
	out.HitBeginTs = in.hit_begin_ts
	out.Last[0].Label = "med"
	out.Last[1].Label = "max"
	out.Last[2].Label = "avg"
	out.Last[3].Label = "rps"
	out.Last[0].Duration = in.hit_end_median
	out.Last[1].Duration = in.hit_end_max
	out.Last[2].Duration = in.hit_end_average
	out.Last[3].Duration = 0
	out.Last[0].Size = in.hit_end_size
	out.Last[1].Size = in.hit_end_size
	out.Last[2].Size = in.hit_end_size
	out.Last[3].Size = in.rps
	out.Current[0].Label = "med"
	out.Current[1].Label = "max"
	out.Current[2].Label = "avg"
	out.Current[3].Label = "rps"
	out.Current[0].Duration, out.Current[1].Duration, out.Current[2].Duration, out.Current[0].Size = in.median.Value(ts)
	out.Current[1].Size = out.Current[0].Size
	out.Current[2].Size = out.Current[0].Size
	_, out.Current[3].Size = in.average.Value(ts)
	return
}
