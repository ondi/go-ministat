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
	Duration     time.Duration
	DurationTs   time.Time
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

func (self *Counter_t) SetState(ts time.Time, duration time.Duration, in int64) {
	if self.state_next != in {
		self.state_next = in
		self.state_next_ts = ts
	}
	if self.state != in && ts.Sub(self.state_next_ts) >= duration {
		self.state = in
		self.state_ts = ts
	}
}

type SetState interface {
	MetricBegin(counter *Counter_t, name string, start time.Time, online int64)
	MetricEnd(counter *Counter_t, name string, start time.Time, online int64, duration time.Duration)
}

type online_limit_t struct {
	limit    int64
	duration time.Duration
}

func NewOnlineLimit(limit int64, duration time.Duration) *online_limit_t {
	return &online_limit_t{limit: limit, duration: duration}
}

func (self *online_limit_t) MetricBegin(counter *Counter_t, name string, start time.Time, online int64) {
	if online >= self.limit {
		counter.SetState(start, self.duration, 1)
	} else {
		counter.SetState(start, self.duration, 0)
	}
}

func (self *online_limit_t) MetricEnd(counter *Counter_t, name string, start time.Time, online int64, duration time.Duration) {
}

type NoState_t struct{}

func (NoState_t) MetricBegin(*Counter_t, string, time.Time, int64) {}

func (NoState_t) MetricEnd(*Counter_t, string, time.Time, int64, time.Duration) {}

type Storage_t struct {
	mx           sync.Mutex
	pages        *unique.Often_t[*Counter_t]
	set_state    SetState
	median_limit int
	median_ttl   time.Duration
}

func NewStorage(limit_pages int, median_limit int, median_ttl time.Duration, set_state SetState) (self *Storage_t) {
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
	counter, _ = self.pages.Add(name, func() *Counter_t {
		return &Counter_t{
			median: NewMedian[time.Duration](self.median_limit, self.median_ttl),
		}
	})
	counter.hits++
	counter.online++
	self.set_state.MetricBegin(counter, name, start, counter.online)
	sampling = counter.sampling
	state = counter.state
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(counter *Counter_t, name string, start time.Time, diff time.Duration, processed int64, errors int64) (sampling int64, duration time.Duration, size int) {
	self.mx.Lock()
	counter.online--
	counter.errors += errors
	counter.processed += processed
	sampling = counter.sampling
	duration, size = counter.median.Add(start.Add(diff), diff, CmpDuration)
	self.set_state.MetricEnd(counter, name, start, counter.online, duration)
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricList(ts time.Time, order Less_t, f func(name string, result Result_t) bool) {
	self.mx.Lock()
	defer self.mx.Unlock()
	self.pages.RangeSort(
		order,
		func(key string, value *Counter_t) bool {
			temp := Result_t{
				Online:    value.online,
				Hits:      value.hits,
				Processed: value.processed,
				Errors:    value.errors,
			}
			temp.Duration, temp.DurationTs, temp.DurationSize = value.median.Median(ts, CmpDuration)
			return f(key, temp)
		},
	)
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
