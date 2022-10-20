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
	Sampling    int64 // reservoir sampling
	Online      int64
	OnlineMax   int64
	Processed   int64
	DurationNum time.Duration
	DurationSum time.Duration
	DurationMax time.Duration
	Status200   int64
	Status400   int64
	Status500   int64
	State       int64
	StateTs     time.Time
	StateNext   int64
	StateNextTs time.Time
}

type Begin_t struct {
	Name    string
	Start   time.Time
	counter *Counter_t
}

func (self *Counter_t) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.Sampling
}

func (self *Counter_t) SetState(ts time.Time, duration time.Duration, in int64) {
	if self.StateNext != in {
		self.StateNext = in
		self.StateNextTs = ts
	}
	if self.State != in && ts.Sub(self.StateNextTs) >= duration {
		self.State = in
		self.StateTs = ts
	}
}

type SetState interface {
	MetricBegin(name string, start time.Time, counter *Counter_t)
	MetricEnd(name string, start time.Time, diff time.Duration, counter *Counter_t)
}

type online_limit_t struct {
	limit    int64
	duration time.Duration
}

func NewOnlineLimit(limit int64, duration time.Duration) *online_limit_t {
	return &online_limit_t{limit: limit, duration: duration}
}

func (self *online_limit_t) MetricBegin(name string, start time.Time, counter *Counter_t) {
	if counter.Online >= self.limit {
		counter.SetState(start, self.duration, 1)
	} else {
		counter.SetState(start, self.duration, 0)
	}
}

func (self *online_limit_t) MetricEnd(name string, start time.Time, diff time.Duration, counter *Counter_t) {
}

type NoState_t struct{}

func (NoState_t) MetricBegin(string, time.Time, *Counter_t) {}

func (NoState_t) MetricEnd(string, time.Time, time.Duration, *Counter_t) {}

type Less_t = cache.Less_t[string, *Counter_t]

func CmpDuration(a, b time.Duration) int {
	return int(a - b)
}

type Storage_t struct {
	mx            sync.Mutex
	timeline      *cache.Cache_t[time.Time, *unique.Often_t[*Counter_t]]
	median        *StorageMedian_t[time.Duration]
	truncate      time.Duration
	evict         Evict
	ts_backlog    int
	limit_pages   int
	set_state     SetState
}

func NewStorage(ts_backlog int, limit_pages int, median_capacity int64, truncate time.Duration, evict Evict, set_state SetState) (self *Storage_t) {
	self = &Storage_t{
		timeline:      cache.New[time.Time, *unique.Often_t[*Counter_t]](),
		median:        NewStorageMedian[time.Duration](limit_pages, median_capacity),
		truncate:      truncate,
		evict:         evict,
		ts_backlog:    ts_backlog,
		limit_pages:   limit_pages,
		set_state:     set_state,
	}
	return
}

func (self *Storage_t) evict_page(page string, value *Counter_t) {
	value.CounterAdd(-value.CounterGet())
	self.evict.MinistatEvict(page, value.DurationSum, value.DurationNum)
}

func (self *Storage_t) MetricBegin(name string, start time.Time) (res Begin_t, sampling int64, state int64) {
	self.mx.Lock()
	if self.timeline.Size() > self.ts_backlog {
		self.timeline.Front().Value.Range(func(page string, value *Counter_t) bool {
			self.evict_page(page, value)
			return true
		})
		self.timeline.Remove(self.timeline.Front().Key)
	}
	it, _ := self.timeline.CreateBack(
		start.Truncate(self.truncate),
		func() *unique.Often_t[*Counter_t] {
			return unique.NewOften(self.limit_pages, self.evict_page)
		},
	)
	res.Name = name
	res.Start = start
	res.counter, _ = it.Value.Add(name, func() *Counter_t { return &Counter_t{} })
	res.counter.Online++
	res.counter.DurationNum++
	if res.counter.Online > res.counter.OnlineMax {
		res.counter.OnlineMax = res.counter.Online
	}
	self.set_state.MetricBegin(name, start, res.counter)
	sampling = res.counter.Sampling
	state = res.counter.State
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(res Begin_t, diff time.Duration, processed int64, status_code int) (sampling int64, median time.Duration) {
	median, _ = self.median.Add(res.Name, diff, CmpDuration)
	
	self.mx.Lock()
	res.counter.Online--
	res.counter.DurationSum += diff
	res.counter.Processed += processed
	if diff > res.counter.DurationMax {
		res.counter.DurationMax = diff
	}
	switch {
	case status_code < 400:
		res.counter.Status200++
	case status_code >= 400 && status_code < 500:
		res.counter.Status400++
	case status_code >= 500:
		res.counter.Status500++
	}
	self.set_state.MetricEnd(res.Name, res.Start, diff, res.counter)
	sampling = res.counter.Sampling
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricListTs(f func(time.Time) bool) {
	self.mx.Lock()
	defer self.mx.Unlock()
	for it := self.timeline.Back(); it != self.timeline.End(); it = it.Prev() {
		if !f(it.Key) {
			return
		}
	}
	return
}

func (self *Storage_t) MetricListRoutes(ts time.Time, order Less_t, f func(name string, counter Counter_t) bool) {
	self.mx.Lock()
	defer self.mx.Unlock()
	if it, ok := self.timeline.Find(ts); ok {
		it.Value.RangeSort(
			order,
			func(key string, value *Counter_t) bool {
				return f(key, *value)
			},
		)
	}
}

func LessHits(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.DurationNum < b.Value.DurationNum
}

func LessProcessed(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.Processed < b.Value.Processed
}

func LessDuration(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Value.DurationSum/a.Value.DurationNum < b.Value.DurationSum/b.Value.DurationNum
}

func LessName(a *cache.Value_t[string, *Counter_t], b *cache.Value_t[string, *Counter_t]) bool {
	return a.Key < b.Key
}
