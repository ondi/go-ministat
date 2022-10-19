//
//
//

package ministat

import "github.com/ondi/go-unique"

type CounterMedian_t[Value_t any] struct {
	Sampling int64 // reservoir sampling
	Median   *Median_t[Value_t]
}

func (self *CounterMedian_t[Valiue_t]) CounterAdd(a int64) {
	self.Sampling += a
}

func (self *CounterMedian_t[Value_t]) CounterGet() int64 {
	return self.Sampling
}

type StorageMedian_t[Value_t any] struct {
	often *unique.Often_t[*CounterMedian_t[Value_t]]
}

func NewStorageMedian[Value_t any](items int, capacity int) (self *StorageMedian_t[Value_t]) {
	self = &StorageMedian_t[Value_t]{
		often: unique.NewOften(items, self.evict_page),
	}
	return
}

func (self *StorageMedian_t[Value_t]) evict_page(page string, value *CounterMedian_t[Value_t]) {

}
