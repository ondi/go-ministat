//
//
//

package ministat

import (
	"sync"
	"time"

	"github.com/ondi/go-unique"
)

type CounterMedian_t[T any] struct {
	Median   *Median_t[T]
	Sampling int64
}

func NewCounterMedian[T any](limit int, ttl time.Duration) *CounterMedian_t[T] {
	return &CounterMedian_t[T]{
		Median: NewMedian[T](limit, ttl),
	}
}

func (self *CounterMedian_t[T]) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *CounterMedian_t[T]) CounterGet() int64 {
	return self.Sampling
}

type StorageMedian_t[T any] struct {
	mx           sync.Mutex
	often        *unique.Often_t[*CounterMedian_t[T]]
	median_ttl   time.Duration
	median_limit int
}

func NewStorageMedian[T any](page_limit int, median_limit int, median_ttl time.Duration) (self *StorageMedian_t[T]) {
	self = &StorageMedian_t[T]{
		often:        unique.NewOften(page_limit, self.evict_page),
		median_ttl:   median_ttl,
		median_limit: median_limit,
	}
	return
}

func (self *StorageMedian_t[T]) Add(ts time.Time, name string, value T, cmp Compare_t[T]) (res T, ok bool) {
	self.mx.Lock()
	it, ok := self.often.Add(name, func() *CounterMedian_t[T] {
		return NewCounterMedian[T](self.median_limit, self.median_ttl)
	})
	res = it.Median.Add(ts, value, cmp)
	self.mx.Unlock()
	return
}

func (self *StorageMedian_t[T]) evict_page(page string, value *CounterMedian_t[T]) {

}
