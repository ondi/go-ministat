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
	Ref         int64 // reservoir sampling
	Online      int64
	OnlineMax   int64
	Processed   int64
	DurationNum time.Duration
	DurationSum time.Duration
	DurationMax time.Duration
	Status200   int64
	Status400   int64
	Status500   int64
	Status000   int64
}

func (self *Counter_t) CounterAdd(a int64) {
	self.Ref += a
}

func (self *Counter_t) CounterGet() int64  {
	return self.Ref
}

type Less_t = cache.Less_t[string, *Counter_t]

type Storage_t struct {
	mx            sync.Mutex
	timeline      *cache.Cache_t[time.Time, *unique.Often_t[*Counter_t]]
	truncate      time.Duration
	views         Views
	limit_backlog int
	limit_items   int
}

func NewStorage(limit_backlog int, limit_items int, truncate time.Duration, views Views) *Storage_t {
	return &Storage_t{
		timeline:      cache.New[time.Time, *unique.Often_t[*Counter_t]](),
		truncate:      truncate,
		views:         views,
		limit_backlog: limit_backlog,
		limit_items:   limit_items,
	}
}

func (self *Storage_t) evict(key string, value *Counter_t) {
	value.CounterAdd(-value.CounterGet())
	self.views.MinistatEvict(key, value.DurationSum, value.DurationNum)
}

func (self *Storage_t) MetricBegin(name string, start time.Time) (counter *Counter_t, current Counter_t) {
	self.mx.Lock()
	if self.timeline.Size() > self.limit_backlog {
		self.timeline.Front().Value.Range(func(key string, value *Counter_t) bool {
			self.evict(key, value)
			return true
		})
		self.timeline.Remove(self.timeline.Front().Key)
	}
	it, _ := self.timeline.CreateBack(
		start.Truncate(self.truncate),
		func() *unique.Often_t[*Counter_t] {
			return unique.NewOften(self.limit_items, self.evict)
		},
	)
	counter, _ = it.Value.Add(name, func() *Counter_t { return &Counter_t{} })
	counter.Online++
	counter.DurationNum++
	if counter.Online > counter.OnlineMax {
		counter.OnlineMax = counter.Online
	}
	current = *counter
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(counter *Counter_t, diff time.Duration, processed int, status_code int) (current Counter_t) {
	self.mx.Lock()
	counter.Online--
	counter.DurationSum += diff
	counter.Processed += int64(processed)
	if diff > counter.DurationMax {
		counter.DurationMax = diff
	}
	switch {
	case status_code >= 200 && status_code < 300:
		counter.Status200++
	case status_code >= 400 && status_code < 500:
		counter.Status400++
	case status_code >= 500:
		counter.Status500++
	default:
		counter.Status000++
	}
	current = *counter
	self.mx.Unlock()
	return
}

func (self *Storage_t) AddDuration(name string, start time.Time, diff time.Duration, processed int, status_code int) {
	c, _ := self.MetricBegin(name, start)
	self.MetricEnd(c, diff, processed, status_code)
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
