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
	recent_begin_ts time.Time
	recent_median   time.Duration
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
	RecentBeginTs time.Time
	Duration      time.Duration
	DurationSize  int
}

type Less_t = cache.Less_t[string, *Counter_t]

func (self *Counter_t) CounterAdd(a int64) {
	self.sampling += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.sampling
}

func NoEvict(page string, value *Counter_t) {}

type Storage_t struct {
	mx           sync.Mutex
	pages        *unique.Often_t[*Counter_t]
	median_ttl   time.Duration
	median_limit int
}

func NewStorage(limit_pages int, median_limit int, median_ttl time.Duration, evict func(page string, value *Counter_t)) (self *Storage_t) {
	self = &Storage_t{
		pages:        unique.NewOften(limit_pages, evict),
		median_ttl:   median_ttl,
		median_limit: median_limit,
	}
	return
}

func (self *Storage_t) HitBegin(name string, begin time.Time) (counter *Counter_t, sampling int64, pending int64) {
	self.mx.Lock()
	counter, _ = self.pages.Add(
		name,
		func(p **Counter_t) {
			*p = &Counter_t{
				median: NewMedian[time.Duration](self.median_limit, self.median_ttl),
			}
		},
	)
	counter.hits++
	counter.pending++
	counter.recent_begin_ts, sampling, pending = begin, counter.sampling, counter.pending
	self.mx.Unlock()
	return
}

func (self *Storage_t) HitEnd(counter *Counter_t, name string, begin time.Time, end time.Time, processed int64, errors int64) (duration time.Duration, size int) {
	self.mx.Lock()
	counter.pending--
	counter.errors += errors
	counter.processed += processed
	counter.recent_median, size = counter.median.Add(end, end.Sub(begin))
	duration = counter.recent_median
	self.mx.Unlock()
	return
}

func (self *Storage_t) HitGet(name string, ts time.Time) (out Result_t, ok bool) {
	self.mx.Lock()
	res, ok := self.pages.Get(name)
	if ok {
		out = ToResult(res, ts)
	}
	self.mx.Unlock()
	return
}

func (self *Storage_t) RangeSort(order Less_t, ts time.Time, f func(name string, res Result_t) bool) {
	self.mx.Lock()
	self.pages.RangeSort(
		order,
		func(key string, value *Counter_t) bool {
			return f(key, ToResult(value, ts))
		},
	)
	self.mx.Unlock()
}

func (self *Storage_t) Range(ts time.Time, f func(name string, res Result_t) bool) {
	self.mx.Lock()
	self.pages.Range(
		func(key string, value *Counter_t) bool {
			return f(key, ToResult(value, ts))
		},
	)
	self.mx.Unlock()
}

func LessHits(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.hits < b.Value.hits
}

func LessProcessed(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.processed < b.Value.processed
}

func LessDuration(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.median.median.Value.Data < b.Value.median.median.Value.Data
}

func LessName(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Key < b.Key
}

func ToResult(in *Counter_t, ts time.Time) Result_t {
	return Result_t{
		Hits:          in.hits,
		Pending:       in.pending,
		Processed:     in.processed,
		Errors:        in.errors,
		RecentBeginTs: in.recent_begin_ts,
		Duration:      in.recent_median,
		DurationSize:  in.median.Evict(ts),
	}
}
