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
	processed    map[string]int64
	errors       map[string]int64
	hit_begin_ts time.Time
	hit_end_ts   time.Time
	hit_end_med  time.Duration
	hit_end_avg  time.Duration
	hit_end_max  time.Duration
	hit_end_size int
	hit_rps      int64
	hits         int64
	pending      int64
	sampling     int64
}

func (self *Counter_t) CounterAdd(a int64) {
	self.sampling += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.sampling
}

type Result_t struct {
	BeginTs      time.Time
	EndTs        time.Time
	GaugeCurrent []Gauge
	GaugeLast    []Gauge
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

func (self *Storage_t[Key_t]) HitBegin(name Key_t, begin time.Time) (counter *Counter_t, sampling int64, pending int64, rps int64) {
	self.mx.Lock()
	counter, _ = self.pages.Add(
		name,
		func(p **Counter_t) {
			*p = &Counter_t{
				median:    NewMedian[time.Duration](self.median_limit, self.median_ttl),
				average:   NewAverage[time.Duration](64, 1*time.Second),
				processed: map[string]int64{},
				errors:    map[string]int64{},
			}
		},
	)
	counter.hits++
	counter.pending++
	_, counter.hit_rps = counter.average.Add(begin, 0)
	counter.hit_begin_ts = begin
	sampling = counter.sampling
	pending = counter.pending
	rps = counter.hit_rps
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitEnd(counter *Counter_t, begin time.Time, end time.Time, processed int64, status string, errors string) {
	self.mx.Lock()
	counter.pending--
	counter.processed[status] += processed
	if len(errors) > 0 {
		counter.errors[errors] += processed
	}
	counter.hit_end_ts = end
	counter.hit_end_med, counter.hit_end_avg, counter.hit_end_max, counter.hit_end_size = counter.median.Add(end, end.Sub(begin))
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitGet(ts time.Time, name Key_t) (out Result_t, ok bool) {
	self.mx.Lock()
	res, ok := self.pages.Get(name)
	if ok {
		out = ToResult(res, ts)
	}
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitRemove(name Key_t) (ok bool) {
	self.mx.Lock()
	ok = self.pages.Del(name)
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) HitRemoveRange(cmp func(Key_t) bool) {
	self.mx.Lock()
	self.pages.Range(
		func(key Key_t, value *Counter_t) bool {
			if cmp(key) {
				self.pages.Del(key)
			}
			return true
		},
	)
	self.mx.Unlock()
	return
}

func (self *Storage_t[Key_t]) RangeSort(ts time.Time, order cache.Less_t[Key_t, *Counter_t], f func(name Key_t, res Result_t) bool) {
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

func LessDuration[Key_t comparable](a *cache.Value_t[Key_t, *Counter_t], b *cache.Value_t[Key_t, *Counter_t]) bool {
	return a.Value.median.median.Value.Data < b.Value.median.median.Value.Data
}

func ToResult(in *Counter_t, ts time.Time) (out Result_t) {
	out.BeginTs = in.hit_begin_ts
	out.EndTs = in.hit_end_ts

	out.GaugeLast = append(out.GaugeLast,
		GaugeInt64_t{Type: "rps", Value: in.hit_rps},
		GaugeInt64_t{Type: "hits", Value: in.hits},
		GaugeInt64_t{Type: "pending", Value: in.pending},
		GaugeDuration_t{Type: "idle", Value: ts.Sub(in.hit_begin_ts)},
		GaugeDuration_t{Type: "latency/med", Value: in.hit_end_med},
		GaugeDuration_t{Type: "latency/avg", Value: in.hit_end_avg},
		GaugeDuration_t{Type: "latency/max", Value: in.hit_end_max},
		GaugeInt64_t{Type: "latency/size", Value: int64(in.hit_end_size)},
	)

	_, rps := in.average.Value(ts)
	med, avg, max, size := in.median.Value(ts)
	out.GaugeCurrent = append(out.GaugeCurrent,
		GaugeInt64_t{Type: "rps", Value: rps},
		GaugeInt64_t{Type: "hits", Value: in.hits},
		GaugeInt64_t{Type: "pending", Value: in.pending},
		GaugeDuration_t{Type: "idle", Value: ts.Sub(in.hit_begin_ts)},
		GaugeDuration_t{Type: "latency/med", Value: med},
		GaugeDuration_t{Type: "latency/avg", Value: avg},
		GaugeDuration_t{Type: "latency/max", Value: max},
		GaugeInt64_t{Type: "latency/size", Value: int64(size)},
	)

	for k, v := range in.processed {
		out.GaugeLast = append(out.GaugeLast, GaugeInt64_t{Type: "processed", Result: k, Value: v})
		out.GaugeCurrent = append(out.GaugeCurrent, GaugeInt64_t{Type: "processed", Result: k, Value: v})
	}
	for k, v := range in.errors {
		out.GaugeLast = append(out.GaugeLast, GaugeInt64_t{Type: "errors", Result: k, Value: v})
		out.GaugeCurrent = append(out.GaugeCurrent, GaugeInt64_t{Type: "errors", Result: k, Value: v})
	}
	return
}
