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
	average         *Average_t[time.Duration]
	hit_begin_ts    time.Time
	hit_end_median  time.Duration
	hit_end_average time.Duration
	sampling        int64
	hits            int64
	pending         int64
	processed       int64
	errors          int64
}

type Result_t struct {
	Hits          int64
	Pending       int64
	Processed     int64
	Errors        int64
	HitBeginTs    time.Time
	HitEndMedian  time.Duration
	HitEndAverage time.Duration
	Median        time.Duration
	Average       time.Duration
	MedianSize    int
	AverageSize   int
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

func (self *Storage_t[Key_t]) HitBegin(name Key_t, begin time.Time) (counter *Counter_t, sampling int64, pending int64) {
	self.mx.Lock()
	counter, _ = self.pages.Add(
		name,
		func(p **Counter_t) {
			*p = &Counter_t{
				median:  NewMedian[time.Duration](self.median_limit, self.median_ttl),
				average: NewAverage[time.Duration](self.median_limit, self.median_ttl),
			}
		},
	)
	counter.hits++
	counter.pending++
	counter.hit_begin_ts, sampling, pending = begin, counter.sampling, counter.pending
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitEnd(counter *Counter_t, name Key_t, begin time.Time, end time.Time, processed int64, errors int64) (median time.Duration, median_size int, average time.Duration, average_size int) {
	self.mx.Lock()
	counter.pending--
	counter.errors += errors
	counter.processed += processed
	diff := end.Sub(begin)
	median, median_size = counter.median.Add(end, diff)
	average, average_size = counter.average.Add(end, diff)
	counter.hit_end_median = median
	counter.hit_end_average = average
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
	out.HitEndMedian = in.hit_end_median
	out.HitEndAverage = in.hit_end_average
	out.Median, out.MedianSize = in.median.Value(ts)
	out.Average, out.AverageSize = in.average.Value(ts)
	return
}
