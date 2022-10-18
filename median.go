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

type ValueCmp_t[Value_t any] func(a, b Value_t) int

type Median_t[Value_t any] struct {
	mx sync.Mutex
	cc *cache.Cache_t[int64, Value_t]
	median *cache.Value_t[int64, Value_t]
	seq int64
	limit int64
	left int64
	right int64
}

func NewMedian[Value_t any](limit int64) (self *Median_t[Value_t]) {
	self = &Median_t[Value_t]{
		cc: cache.New[int64, Value_t](),
		limit: limit,
	}
	self.median = self.cc.End()
	return
}

func (self *Median_t[Value_t]) Add(value Value_t, cmp ValueCmp_t[Value_t]) (res Value_t) {
	self.mx.Lock()
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	var prev_less_than_median bool
	fmt.Fprintf(os.Stderr, "###########: value=%v, median=%v, left=%v, right=%v, values=%v\n", value, self.median.Value, self.left, self.right, self.Values())
	it, ok := self.cc.CreateFront(self.seq, func() Value_t{return value})
	if !ok {
		// тут нужно знать из какой половины списка старый элемент
		// чтобы понимать надо ли менять число элементов слева и справа медианы или оставить как есть
		if cmp(it.Value, self.median.Value) <= 0 {
			prev_less_than_median = true
		}
		fmt.Fprintf(os.Stderr, "OVERWRITE VALUE: old=%v, new=%v, prev_less=%v, median=%v, left=%v, right=%v, values=%v\n", it.Value, value, prev_less_than_median, self.median.Value, self.left, self.right, self.Values())
		if it == self.median {
			self.median = self.median.Next()
			prev_less_than_median = true
			self.left++
			self.right--
			fmt.Fprintf(os.Stderr, "OVERWRITE MEDIAN: old=%v, new=%v, prev_less=%v, median=%v, left=%v, right=%v, values=%v\n", it.Value, value, prev_less_than_median, self.median.Value, self.left, self.right, self.Values())
		}
		it.Value = value
	}
	median_passed := self.insert_value(it, cmp)
	self.set_median(it, median_passed, ok, prev_less_than_median)
	res = self.median.Value
	self.mx.Unlock()
	return
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
	for it := self.cc.Front(); it != self.cc.End(); it = it.Next() {
		res = append(res, it.Value)
	}
	return
}

func (self *Median_t[Value_t]) RealMedian() (it *cache.Value_t[int64, Value_t]) {
	half := self.cc.Size() / 2
	for it = self.cc.Front(); half > 0; it = it.Next() {
		half--
	}
	return
}

func (self *Median_t[Value_t]) set_median(it *cache.Value_t[int64, Value_t], median_passed bool, inserted bool, prev_less_than_median bool) {
	
	fmt.Fprintf(os.Stderr, "SET MEDIAN BEGIN: inserted=%v, passed=%v, before=%v, left=%v, right=%v, median=%v, values=%v\n", inserted, median_passed, prev_less_than_median, self.left, self.right, self.median.Value, self.Values())
	
	if median_passed {
		if inserted {
			self.right++
		} else if prev_less_than_median {
			self.left--
			self.right++
		}
	} else if self.cc.Size() > 1 {
		if inserted {
			self.left++
		} else if prev_less_than_median == false {
			self.left++
			self.right--
		}
	} else {
		self.median = it
	}

	fmt.Fprintf(os.Stderr, "SET MEDIAN   END: inserted=%v, passed=%v, before=%v, left=%v, right=%v, median=%v, values=%v\n", inserted, median_passed, prev_less_than_median, self.left, self.right, self.median.Value, self.Values())

	if self.right < self.left - 1 {
		self.median = self.median.Prev()
		self.left--
		self.right++
		fmt.Fprintf(os.Stderr, "MEDIAN MOVE PREV: left=%v, right=%v, median=%v, values=%v\n", self.left, self.right, self.median.Value, self.Values())
	} else if self.left < self.right - 1 {
		self.median = self.median.Next()
		self.left++
		self.right--
		fmt.Fprintf(os.Stderr, "MEDIAN MOVE NEXT: left=%v, right=%v, median=%v, values=%v\n", self.left, self.right, self.median.Value, self.Values())
	}
	it = self.cc.Front()
	for i := int64(0); i < self.left; i++ {
		it = it.Next()
	}
	if it.Key != self.median.Key {
		panic(fmt.Sprintf("MEDIAN CHECK: left=%v, right=%v, check=%v, median=%v", self.left, self.right, it, self.median))
	}
}

func (self *Median_t[Value_t]) insert_value(it *cache.Value_t[int64, Value_t], cmp ValueCmp_t[Value_t]) (median_passed bool) {
	for at := self.cc.Front(); at != self.cc.End(); at = at.Next() {
		if cmp(it.Value, at.Value) < 0 {
			cache.CutList(it)
			cache.SetPrev(it, at)
			return
		}
		if at == self.median {
			median_passed = true
		}
	}
	cache.CutList(it)
	cache.SetPrev(it, self.cc.End())
	return
}
