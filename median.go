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
	on_left int
	on_right int
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
	if self.seq == self.median.Key {
		self.mx.Unlock()
		fmt.Fprintf(os.Stderr, "MEDIAN OVERVRITE SKIPPED\n")
		return
	}
	it, ok := self.cc.PushFront(self.seq, func() Value_t{return value})
	if !ok {
		fmt.Fprintf(os.Stderr, "OVERWRITE old=%v, new=%v, median=%v, left=%v, right=%v\n", it, value, self.median, self.on_left, self.on_right)
		less_before := less(it, self.median)
		it.Value = value
		less_after := less(it, self.median)
		if less_before == true && less_after == false {
			self.on_left--
			self.on_right++
		}
		if less_before == false && less_after == true {
			self.on_left++
			self.on_right--
		}
		fmt.Fprintf(os.Stderr, "OVERWRITE less_before=%v, less_after=%v, left=%v, right=%v\n", less_before, less_after, self.on_left, self.on_right)
	} else {
		if self.cc.Size() == 1 {
			self.median = it
		} else if less(it, self.median) {
			self.on_left++
			fmt.Fprintf(os.Stderr, "VALUE LESS value=%v, median=%v, left=%v, right=%v\n", it.Value, self.median, self.on_left, self.on_right)
		} else {
			self.on_right++
			fmt.Fprintf(os.Stderr, "VALUE GREATER value=%v, median=%v, left=%v, right=%v\n", it.Value, self.median, self.on_left, self.on_right)
		}
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

func (self *Median_t[Value_t]) Values() (res []Value_t) {
	for it := self.cc.Front(); it != self.cc.End(); it=it.Next() {
		res = append(res, it.Value)
	}
	return
}

func (self *Median_t[Value_t]) RealMedian() (it *cache.Value_t[int64, Value_t]) {
	half := self.cc.Size() / 2
	for it = self.cc.Front(); half>0; it=it.Next() {
		half--
	}
	return
}

func (self *Median_t[Value_t]) SetMedian() {
	fmt.Fprintf(os.Stderr, "### MEDIAN: Values: %v, Median: %v, left=%v, right=%v, \n", self.Values(), self.median, self.on_left, self.on_right)
	if self.on_left < self.on_right - 1 {
		self.on_left++
		self.on_right--
		self.median = self.median.Next()
		fmt.Fprintf(os.Stderr, "MOVED NEXT TO: %v, left=%v, right=%v\n", self.median, self.on_left, self.on_right)
	} else if self.on_left - 1 > self.on_right {
		self.on_left--
		self.on_right++
		self.median = self.median.Prev()
		fmt.Fprintf(os.Stderr, "MOVED PREV TO: %v, left=%v, right=%v\n", self.median, self.on_left, self.on_right)
	}
	it := self.cc.Front()
	for i := 0; i < self.on_left; i++ {
		it = it.Next()
	}
	if it.Value != self.median.Value {
		panic(fmt.Sprintf("ERROR: %v %v %v %v\n", self.on_left, self.on_right, it.Value, self.median))
	}
}

func (self *Median_t[Value_t]) SortValueFront(it *cache.Value_t[int64, Value_t], less cache.Less_t[int64, Value_t]) {
	for v := self.cc.Front().Next(); v != self.cc.End(); v = v.Next() {
		if less(it, v) {
			cache.CutList(it)
			cache.SetPrev(it, v)
			self.SetMedian()
			return
		}
	}
	cache.CutList(it)
	cache.SetPrev(it, self.cc.End())
	self.SetMedian()
}
