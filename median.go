//
//
//

package ministat

import (
	"fmt"
	"os"
	"sync"

	"github.com/ondi/go-cache"
)

type Median_t[Value_t comparable] struct {
	mx sync.Mutex
	cc *cache.Cache_t[int64, Value_t]
	median *cache.Value_t[int64, Value_t]
	seq int64
	limit int64
}

func NewMedian[Value_t comparable](limit int64) (self *Median_t[Value_t]) {
	self = &Median_t[Value_t]{
		cc: cache.New[int64, Value_t](),
		limit: limit,
	}
	self.median = self.cc.End()
	return
}

func (self *Median_t[Value_t]) Add(value Value_t, less cache.Less_t[int64, Value_t]) {
	self.mx.Lock()
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	it, ok := self.cc.PushFront(self.seq, func() Value_t{return value})
	if !ok {
		it.Value = value
	}
	self.SortValueFront(it, less)
	self.mx.Unlock()
}

func (self *Median_t[Value_t]) Median() (res Value_t) {
	self.mx.Lock()
	res = self.median.Value
	self.mx.Unlock()
	return
}

func (self *Median_t[Value_t]) Size() (res int) {
	self.mx.Lock()
	res = self.cc.Size()
	self.mx.Unlock()
	return
}

func (self *Median_t[Value_t]) Range(f func(key int64, value Value_t) bool) {
	self.mx.Lock()
	defer self.mx.Unlock()
	for it := self.cc.Front(); it != self.cc.End(); it = it.Next() {
		if f(it.Key, it.Value) == false {
			return
		}
	}
}

func (self *Median_t[Value_t]) SortValueFront(it *cache.Value_t[int64, Value_t], less cache.Less_t[int64, Value_t]) {
	var median_passed bool
	fmt.Fprintf(os.Stderr, "### INPUT: %v\n", it.Value)
	for v := self.cc.Front().Next(); v != self.cc.End(); v = v.Next() {
		fmt.Fprintf(os.Stderr, "CHECK: %v %v\n", it.Value, v.Value)
		if v == self.median {
			median_passed = true
		}
		if less(it, v) {
			cache.CutList(it)
			cache.SetPrev(it, v)
			if median_passed {
				self.median = self.median.Prev()
				fmt.Fprintf(os.Stderr, "MEDIAN PASSED, MOVED TO: %v\n", self.median.Value)
			}
			return
		}
	}
	self.median = self.median.Prev()
	fmt.Fprintf(os.Stderr, "DEFAULT MEDIAN TO: %v\n", self.median.Value)
	cache.CutList(it)
	cache.SetPrev(it, self.cc.End())
}
