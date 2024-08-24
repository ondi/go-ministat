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
	median       *Median_t[time.Duration]
	average      *Average_t[time.Duration] // RPS
	hit_begin_ts time.Time
	hit_end_med  time.Duration
	hit_end_max  time.Duration
	hit_end_avg  time.Duration
	hit_end_size int
	rps          int64
	sampling     int64
	hits         int64
	pending      int64
	processed    int64
	errors       int64
}

type Result_t struct {
	Hits            int64
	Pending         int64
	Processed       int64
	Errors          int64
	HitBeginTs      time.Time
	GaugeLast       [6]Gauge_t
	GaugeCurrent    [2]Gauge_t
	DurationLast    [3]Duration_t
	DurationCurrent [3]Duration_t
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

func (self *Storage_t[Key_t]) HitBegin(name Key_t, begin time.Time) (counter *Counter_t, sampling int64, g [3]Gauge_t) {
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
	g[0] = Gauge_t{Label: "pending", Value: counter.pending}
	g[1] = Gauge_t{Label: "hits", Value: counter.hits}
	g[2] = Gauge_t{Label: "rps", Value: counter.rps}
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitEnd(counter *Counter_t, begin time.Time, end time.Time, processed int64, errors int64) (g [4]Gauge_t, d [3]Duration_t) {
	self.mx.Lock()
	counter.pending--
	counter.errors += errors
	counter.processed += processed
	counter.hit_end_med, counter.hit_end_max, counter.hit_end_avg, counter.hit_end_size = counter.median.Add(end, end.Sub(begin))
	d[0] = Duration_t{Label: "med", Value: counter.hit_end_med}
	d[1] = Duration_t{Label: "max", Value: counter.hit_end_max}
	d[2] = Duration_t{Label: "avg", Value: counter.hit_end_avg}
	g[0] = Gauge_t{Label: "pending", Value: counter.pending}
	g[1] = Gauge_t{Label: "errors", Value: counter.errors}
	g[2] = Gauge_t{Label: "processed", Value: counter.processed}
	g[3] = Gauge_t{Label: "size", Value: int64(counter.hit_end_size)}
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
	out.HitBeginTs = in.hit_begin_ts

	out.GaugeLast[0] = Gauge_t{Label: "pending", Value: in.pending}
	out.GaugeLast[1] = Gauge_t{Label: "hits", Value: in.hits}
	out.GaugeLast[2] = Gauge_t{Label: "rps", Value: in.rps}
	out.GaugeLast[3] = Gauge_t{Label: "errors", Value: in.errors}
	out.GaugeLast[4] = Gauge_t{Label: "processed", Value: in.processed}
	out.GaugeLast[5] = Gauge_t{Label: "size", Value: int64(in.hit_end_size)}

	out.DurationLast[0] = Duration_t{Label: "med", Value: in.hit_end_med}
	out.DurationLast[1] = Duration_t{Label: "max", Value: in.hit_end_max}
	out.DurationLast[2] = Duration_t{Label: "avg", Value: in.hit_end_avg}

	var size int
	out.DurationCurrent[0].Label = "med"
	out.DurationCurrent[1].Label = "max"
	out.DurationCurrent[2].Label = "avg"
	out.DurationCurrent[0].Value, out.DurationCurrent[1].Value, out.DurationCurrent[2].Value, size = in.median.Value(ts)

	out.GaugeCurrent[0] = Gauge_t{Label: "size", Value: int64(size)}
	out.GaugeCurrent[1].Label = "rps"
	_, out.GaugeCurrent[1].Value = in.average.Value(ts)
	return
}
