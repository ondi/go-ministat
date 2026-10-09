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
	buckets     int
}

func NewAverage[T Number](buckets int, ttl time.Duration) (self *Average_t[T]) {
	self = &Average_t[T]{
		cx:       cache.New[time.Time, AverageMapped_t[T]](),
		ttl:      ttl,
		truncate: ttl / time.Duration(buckets),
		buckets:  buckets,
	}
	return
}

func (self *Average_t[T]) Add(ts time.Time, data T) (T, int64) {
	self.Evict(ts)
	self.cx.CreateBack(
		ts.Add(self.ttl).Truncate(self.truncate),
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
	return self.total_sum / self.total_count, int64(self.total_count)
}

func (self *Average_t[T]) Evict(ts time.Time) int {
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if self.cx.Size() > self.buckets || ts.Before(it.Key) == false {
			self.total_sum -= it.Value.Data
			self.total_count -= it.Value.Count
			self.cx.Remove(it.Key)
		} else {
			return self.cx.Size()
		}
	}
	return 0
}

func (self *Average_t[T]) Value(ts time.Time) (value T, count int64) {
	self.Evict(ts)
	if count = int64(self.total_count); count > 0 {
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
