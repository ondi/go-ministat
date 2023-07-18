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
	median        *Median_t[time.Duration]
	state_ts      time.Time
	state_next_ts time.Time
	begin_last_ts time.Time
	last_median   time.Duration
	sampling      int64
	online        int64
	hits          int64
	processed     int64
	errors        int64
	state         int64
	state_next    int64
}

type Result_t struct {
	Online       int64
	Hits         int64
	Processed    int64
	Errors       int64
	BeginLastTs  time.Time
	Duration     time.Duration
	DurationSize int
}

type Less_t = cache.Less_t[string, *Counter_t]

func CmpDuration(a, b time.Duration) int {
	return int(a - b)
}

func (self *Counter_t) CounterAdd(a int64) {
	self.sampling += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.sampling
}

func (self *Counter_t) SetState(ts time.Time, delay time.Duration, in int64) {
	if self.state_next != in {
		self.state_next = in
		self.state_next_ts = ts
	}
	if self.state != in && ts.Sub(self.state_next_ts) >= delay {
		self.state = in
		self.state_ts = ts
	}
}

type StateSetter interface {
	SetState(counter *Counter_t, ts time.Time, online int64)
}

type online_limit_t struct {
	limit    int64
	duration time.Duration
}

func NewOnlineLimit(limit int64, duration time.Duration) *online_limit_t {
	return &online_limit_t{limit: limit, duration: duration}
}

func (self *online_limit_t) SetState(counter *Counter_t, ts time.Time, online int64) {
	if online >= self.limit {
		counter.SetState(ts, self.duration, 1)
	} else {
		counter.SetState(ts, self.duration, 0)
	}
}

type NoState_t struct{}

func (NoState_t) SetState(*Counter_t, time.Time, int64) {}

type Storage_t struct {
	mx           sync.Mutex
	pages        *unique.Often_t[*Counter_t]
	set_state    StateSetter
	median_limit int
	median_ttl   time.Duration
}

func NewStorage(limit_pages int, median_limit int, median_ttl time.Duration, set_state StateSetter) (self *Storage_t) {
	self = &Storage_t{
		pages:        unique.NewOften(limit_pages, self.evict_page),
		set_state:    set_state,
		median_limit: median_limit,
		median_ttl:   median_ttl,
	}
	return
}

func (self *Storage_t) evict_page(page string, value *Counter_t) {

}

func (self *Storage_t) MetricBegin(name string, start time.Time) (counter *Counter_t, sampling int64, state int64) {
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
	counter.online++
	self.set_state.SetState(counter, start, counter.online)
	counter.begin_last_ts, sampling, state = start, counter.sampling, counter.state
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(counter *Counter_t, name string, start time.Time, end time.Time, processed int64, errors int64) (duration time.Duration, size int) {
	self.mx.Lock()
	counter.online--
	counter.errors += errors
	counter.processed += processed
	counter.last_median, size = counter.median.Add(end, end.Sub(start), CmpDuration)
	duration = counter.last_median
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricGet(name string, ts time.Time) (out Result_t, ok bool) {
	self.mx.Lock()
	res, ok := self.pages.Get(name)
	if ok {
		out = to_result(res, ts)
	}
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricListSort(order Less_t, ts time.Time, f func(name string, res Result_t) bool) {
	self.mx.Lock()
	self.pages.RangeSort(
		order,
		func(key string, value *Counter_t) bool {
			return f(key, to_result(value, ts))
		},
	)
	self.mx.Unlock()
}

func (self *Storage_t) MetricList(ts time.Time, f func(name string, res Result_t) bool) {
	self.mx.Lock()
	self.pages.Range(
		func(key string, value *Counter_t) bool {
			return f(key, to_result(value, ts))
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

func to_result(in *Counter_t, ts time.Time) Result_t {
	return Result_t{
		Online:       in.online,
		Hits:         in.hits,
		Processed:    in.processed,
		Errors:       in.errors,
		BeginLastTs:  in.begin_last_ts,
		Duration:     in.last_median,
		DurationSize: in.median.Evict(ts, CmpDuration),
	}
}
