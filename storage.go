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

func (self *Counter_t) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *Counter_t) CounterGet() int64  {
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

type online_limit_t struct {
	limit    int64
	duration time.Duration
}

func NewOnlineLimit(limit int64, duration time.Duration) *online_limit_t {
	return &online_limit_t{limit: limit, duration: duration}
}

func (self *online_limit_t) SetState(ts time.Time, in *Counter_t) {
	if in.Online >= self.limit {
		in.SetState(ts, self.duration, 1)
	} else {
		in.SetState(ts, self.duration, 0)
	}
}

type SetState_t func(time.Time, *Counter_t)
type Less_t = cache.Less_t[string, *Counter_t]

func NoState(time.Time, *Counter_t) {}

type Storage_t struct {
	mx            sync.Mutex
	timeline      *cache.Cache_t[time.Time, *unique.Often_t[*Counter_t]]
	truncate      time.Duration
	evict         Evict
	limit_backlog int
	limit_items   int
	set_state     SetState_t
}

func NewStorage(backlog int, items int, truncate time.Duration, evict Evict, set_state SetState_t) (self *Storage_t) {
	self = &Storage_t{
		timeline:      cache.New[time.Time, *unique.Often_t[*Counter_t]](),
		truncate:      truncate,
		evict:         evict,
		limit_backlog: backlog,
		limit_items:   items,
		set_state:     set_state,
	}
	return
}

func (self *Storage_t) evict_page(page string, value *Counter_t) {
	value.CounterAdd(-value.CounterGet())
	self.evict.MinistatEvict(page, value.DurationSum, value.DurationNum)
}

func (self *Storage_t) MetricBegin(name string, start time.Time) (counter *Counter_t, current Counter_t) {
	self.mx.Lock()
	if self.timeline.Size() > self.limit_backlog {
		self.timeline.Front().Value.Range(func(page string, value *Counter_t) bool {
			self.evict_page(page, value)
			return true
		})
		self.timeline.Remove(self.timeline.Front().Key)
	}
	it, _ := self.timeline.CreateBack(
		start.Truncate(self.truncate),
		func() *unique.Often_t[*Counter_t] {
			return unique.NewOften(self.limit_items, self.evict_page)
		},
	)
	counter, _ = it.Value.Add(name, func() *Counter_t { return &Counter_t{} })
	counter.Online++
	counter.DurationNum++
	if counter.Online > counter.OnlineMax {
		counter.OnlineMax = counter.Online
	}
	self.set_state(start, counter)
	current = *counter
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(counter *Counter_t, diff time.Duration, processed int64, status_code int) (current Counter_t) {
	self.mx.Lock()
	counter.Online--
	counter.DurationSum += diff
	counter.Processed += processed
	if diff > counter.DurationMax {
		counter.DurationMax = diff
	}
	switch {
	case status_code < 400:
		counter.Status200++
	case status_code >= 400 && status_code < 500:
		counter.Status400++
	case status_code >= 500:
		counter.Status500++
	}
	current = *counter
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
