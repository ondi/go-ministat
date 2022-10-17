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

type Compare_t[Value_t comparable] func (a, b Value_t) int

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

func (self *Median_t[Value_t]) Add(value Value_t, cmp Compare_t[Value_t]) {
	self.mx.Lock()
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	fmt.Fprintf(os.Stderr, "### INPUT: %v\n", value)
	it, ok := self.cc.PushFront(self.seq, func() Value_t{return value})
	if !ok {
		it.Value = value
	}
	self.SortValueFront(it, cmp)
	fmt.Fprintf(os.Stderr, "MIDIAN: %v, Values: %v\n", self.median.Value, self.Values())
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

func (self *Median_t[Value_t]) SetMedian(out *cache.Value_t[int64, Value_t], count int) {
	for ; count < self.cc.Size() / 2; count++ {
		fmt.Fprintf(os.Stderr, "REWIND: %v %v\n", count, self.cc.Size())
		out = out.Next()
	}
	self.median = out
}

func (self *Median_t[Value_t]) SortValueFront(it *cache.Value_t[int64, Value_t], cmp Compare_t[Value_t]) (out *cache.Value_t[int64, Value_t], count int) {
	for out = self.cc.Front().Next(); out != self.cc.End(); out = out.Next() {
		count++
		fmt.Fprintf(os.Stderr, "COUNT: %v, size=%v, value=%v\n", count, self.cc.Size(), out.Value)
		if count == (self.cc.Size() - 1) / 2 {
			fmt.Fprintf(os.Stderr, "MEDIAN FIRED: %v\n", out.Value)
			self.median = out
		}
		if cmp(it.Value, out.Value) < 0 {
			cache.CutList(it)
			cache.SetPrev(it, out)
			self.SetMedian(out, count)
			return
		}
	}
	cache.CutList(it)
	cache.SetPrev(it, self.cc.End())
	if self.cc.Size() == 1 {
		self.median = it
	}
	return
}
