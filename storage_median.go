//
//
//

package ministat

import (
	"sync"
	"time"

	"github.com/ondi/go-unique"
)

type CounterMedian_t[Measure_t any] struct {
	Sampling int64 // reservoir sampling
	Median   *Median_t[Measure_t]
}

func NewCounterMedian[Measure_t any](capacity int64) *CounterMedian_t[Measure_t] {
	return &CounterMedian_t[Measure_t]{
		Median: NewMedian[Measure_t](capacity),
	}
}

func (self *CounterMedian_t[Valiue_t]) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *CounterMedian_t[Measure_t]) CounterGet() int64 {
	return self.Sampling
}

type StorageMedian_t[Measure_t any] struct {
	mx              sync.Mutex
	often           *unique.Often_t[*CounterMedian_t[Measure_t]]
	median_capacity int64
}

func NewStorageMedian[Measure_t any](limit_pages int, median_capacity int64) (self *StorageMedian_t[Measure_t]) {
	self = &StorageMedian_t[Measure_t]{
		often:           unique.NewOften(limit_pages, self.evict_page),
		median_capacity: median_capacity,
	}
	return
}

func (self *StorageMedian_t[Measure_t]) Add(name string, ts time.Time, value Measure_t, cmp Compare_t[Measure_t]) (res Measure_t, ok bool) {
	self.mx.Lock()
	it, ok := self.often.Add(name, func() *CounterMedian_t[Measure_t] { return NewCounterMedian[Measure_t](self.median_capacity) })
	res = it.Median.Add(ts, value, cmp)
	self.mx.Unlock()
	return
}

func (self *StorageMedian_t[Measure_t]) evict_page(page string, value *CounterMedian_t[Measure_t]) {

}
