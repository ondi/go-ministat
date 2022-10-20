//
//
//

package ministat

import (
	"sync"

	"github.com/ondi/go-unique"
)

type CounterMedian_t[Value_t any] struct {
	Sampling int64 // reservoir sampling
	Median   *Median_t[Value_t]
}

func NewCounterMedian[Value_t any](capacity int64) *CounterMedian_t[Value_t] {
	return &CounterMedian_t[Value_t]{
		Median: NewMedian[Value_t](capacity),
	}
}

func (self *CounterMedian_t[Valiue_t]) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *CounterMedian_t[Value_t]) CounterGet() int64 {
	return self.Sampling
}

type StorageMedian_t[Value_t any] struct {
	mx              sync.Mutex
	often           *unique.Often_t[*CounterMedian_t[Value_t]]
	median_capacity int64
}

func NewStorageMedian[Value_t any](limit_pages int, median_capacity int64) (self *StorageMedian_t[Value_t]) {
	self = &StorageMedian_t[Value_t]{
		often:           unique.NewOften(limit_pages, self.evict_page),
		median_capacity: median_capacity,
	}
	return
}

func (self *StorageMedian_t[Value_t]) Add(name string, value Value_t, cmp Compare_t[Value_t]) (res Value_t, ok bool) {
	self.mx.Lock()
	it, ok := self.often.Add(name, func() *CounterMedian_t[Value_t] { return NewCounterMedian[Value_t](self.median_capacity) })
	res = it.Median.Add(value, cmp)
	self.mx.Unlock()
	return
}

func (self *StorageMedian_t[Value_t]) evict_page(page string, value *CounterMedian_t[Value_t]) {

}
