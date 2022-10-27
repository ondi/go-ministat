//
//
//

package ministat

import (
	"sync"

	"github.com/ondi/go-unique"
)

type CounterMedian_t[Measure_t any] struct {
	Median   *Median_t[Measure_t]
	Sampling int64
}

func NewCounterMedian[Measure_t any](limit int) *CounterMedian_t[Measure_t] {
	return &CounterMedian_t[Measure_t]{
		Median: NewMedian[Measure_t](limit),
	}
}

func (self *CounterMedian_t[Valiue_t]) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *CounterMedian_t[Measure_t]) CounterGet() int64 {
	return self.Sampling
}

type StorageMedian_t[Measure_t any] struct {
	mx           sync.Mutex
	often        *unique.Often_t[*CounterMedian_t[Measure_t]]
	median_limit int
}

func NewStorageMedian[Measure_t any](page_limit int, median_limit int) (self *StorageMedian_t[Measure_t]) {
	self = &StorageMedian_t[Measure_t]{
		often:        unique.NewOften(page_limit, self.evict_page),
		median_limit: median_limit,
	}
	return
}

func (self *StorageMedian_t[Measure_t]) Add(name string, value Measure_t, cmp Compare_t[Measure_t]) (res Measure_t, ok bool) {
	self.mx.Lock()
	it, ok := self.often.Add(name, func() *CounterMedian_t[Measure_t] {
		return NewCounterMedian[Measure_t](self.median_limit)
	})
	res = it.Median.Add(value, cmp)
	self.mx.Unlock()
	return
}

func (self *StorageMedian_t[Measure_t]) evict_page(page string, value *CounterMedian_t[Measure_t]) {

}
