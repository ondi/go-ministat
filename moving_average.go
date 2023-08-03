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
	Count int
}

type Average_t[T Number] struct {
	cx          *cache.Cache_t[time.Time, AverageMapped_t[T]]
	truncate    time.Duration
	total_sum   T
	total_count int
	limit       int
}

func NewAverage[T Number](limit int, ttl time.Duration) (self *Average_t[T]) {
	self = &Average_t[T]{
		cx:       cache.New[time.Time, AverageMapped_t[T]](),
		limit:    limit,
		truncate: ttl / time.Duration(limit),
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
	return self.total_sum / T(self.total_count), self.cx.Size()
}

func (self *Average_t[T]) Evict(ts time.Time) int {
	return 0
}
