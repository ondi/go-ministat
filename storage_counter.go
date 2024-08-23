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
	Hits            int64
	Pending         int64
	Processed       int64
	Errors          int64
	HitBeginTs      time.Time
	DurationLast    [3]Duration_t
	DurationCurrent [3]Duration_t
	GaugeLast       [1]Gauge_t
	GaugeCurrent    [1]Gauge_t
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

func (self *Storage_t[Key_t]) HitBegin(name Key_t, begin time.Time) (counter *Counter_t, sampling int64, pending int64, g [1]Gauge_t) {
	self.mx.Lock()
	counter, _ = self.pages.Add(
		name,
		func(p **Counter_t) {
			*p = &Counter_t{
				median:  NewMedian[time.Duration](self.median_limit, self.median_ttl),
				average: NewAverage[time.Duration](64, 1*time.Second),
			}
		},
	)
	counter.hits++
	counter.pending++
	_, counter.rps = counter.average.Add(begin, 0)
	counter.hit_begin_ts = begin
	sampling = counter.sampling
	pending = counter.pending
	g[0].Label = "rps"
	g[0].Value = counter.rps
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
	out[0].Value = counter.hit_end_median
	out[1].Value = counter.hit_end_max
	out[2].Value = counter.hit_end_average
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

	out.DurationLast[0].Label = "med"
	out.DurationLast[1].Label = "max"
	out.DurationLast[2].Label = "avg"
	out.DurationLast[0].Value = in.hit_end_median
	out.DurationLast[1].Value = in.hit_end_max
	out.DurationLast[2].Value = in.hit_end_average
	out.DurationLast[0].Size = in.hit_end_size
	out.DurationLast[1].Size = in.hit_end_size
	out.DurationLast[2].Size = in.hit_end_size

	out.GaugeLast[0].Label = "rps"
	out.GaugeLast[0].Value = in.rps

	out.DurationCurrent[0].Label = "med"
	out.DurationCurrent[1].Label = "max"
	out.DurationCurrent[2].Label = "avg"
	out.DurationCurrent[0].Value, out.DurationCurrent[1].Value, out.DurationCurrent[2].Value, out.DurationCurrent[0].Size = in.median.Value(ts)
	out.DurationCurrent[1].Size = out.DurationCurrent[0].Size
	out.DurationCurrent[2].Size = out.DurationCurrent[0].Size

	out.GaugeCurrent[0].Label = "rps"
	_, out.GaugeCurrent[0].Value = in.average.Value(ts)
	return
}
