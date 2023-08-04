//
//
//

package ministat

import (
	"time"

	"github.com/ondi/go-cache"
)

type AverageMapped_t[T Number] struct {
	Data  T
	Count T
}

type Average_t[T Number] struct {
	cx          *cache.Cache_t[time.Time, AverageMapped_t[T]]
	ttl         time.Duration
	truncate    time.Duration
	total_sum   T
	total_count T
	limit       int
}

func NewAverage[T Number](limit int, ttl time.Duration) (self *Average_t[T]) {
	self = &Average_t[T]{
		cx:       cache.New[time.Time, AverageMapped_t[T]](),
		ttl:      ttl,
		truncate: ttl / time.Duration(limit),
		limit:    limit,
	}
	return
}

func (self *Average_t[T]) Add(ts time.Time, data T) (T, int) {
	self.Evict(ts)
	self.cx.CreateBack(
		ts.Truncate(self.truncate),
		func(p *AverageMapped_t[T]) {
			p.Data += data
			p.Count++
		},
		func(p *AverageMapped_t[T]) {
			p.Data += data
			p.Count++
		},
	)
	self.total_sum += data
	self.total_count++
	return self.total_sum / self.total_count, self.cx.Size()
}

func (self *Average_t[T]) Evict(ts time.Time) int {
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if self.cx.Size() <= self.limit && ts.Sub(it.Key) < self.ttl {
			return self.cx.Size()
		}
		self.total_sum -= it.Value.Data
		self.total_count -= it.Value.Count
		self.cx.Remove(it.Key)
	}
	return 0
}

func (self *Average_t[T]) Value(ts time.Time) (value T, size int) {
	if size = self.Evict(ts); size > 0 {
		value = self.total_sum / self.total_count
	}
	return
}

func (self *Average_t[T]) range_test(ts time.Time, f func(key time.Time, value AverageMapped_t[T]) bool) {
	self.Evict(ts)
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if f(it.Key, it.Value) == false {
			return
		}
	}
}
