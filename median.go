//
//
//

package ministat

import (
	"sync"

	"github.com/ondi/go-cache"
)

type Compare_t[Value_t any] func(a, b Value_t) int

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

func (self *Median_t[Value_t]) Add(value Value_t, cmp Compare_t[Value_t]) (res Value_t) {
	self.mx.Lock()
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	var prev_less_than_median bool
	it, inserted := self.cc.CreateFront(self.seq, func() Value_t{return value})
	if !inserted {
		// тут нужно знать из какой половины списка старый элемент
		// чтобы скорректировать число элементов слева и справа медианы или оставить как есть
		if cmp(it.Value, self.median.Value) <= 0 {
			prev_less_than_median = true
		}
		if it == self.median {
			self.median = self.median.Next()
			prev_less_than_median = true
			self.left++
			self.right--
		}
		it.Value = value
	}
	median_passed := self.move_value(it, cmp)
	self.set_median(it, median_passed, inserted, prev_less_than_median)
	res = self.median.Value
	self.mx.Unlock()
	return
}

func (self *Median_t[Value_t]) move_value(it *cache.Value_t[int64, Value_t], cmp Compare_t[Value_t]) (median_passed bool) {
	for at := self.cc.Front(); at != self.cc.End(); at = at.Next() {
		if cmp(it.Value, at.Value) < 0 {
			cache.CutList(it)
			cache.SetPrev(it, at)
			return
		}
		median_passed = median_passed || at == self.median
	}
	cache.CutList(it)
	cache.SetPrev(it, self.cc.End())
	return
}

func (self *Median_t[Value_t]) set_median(it *cache.Value_t[int64, Value_t], median_passed bool, inserted bool, prev_less_than_median bool) {
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
	if self.right < self.left - 1 {
		self.median = self.median.Prev()
		self.left--
		self.right++
	} else if self.left < self.right - 1 {
		self.median = self.median.Next()
		self.left++
		self.right--
	}
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

func (self *Median_t[Value_t]) DebugLR() (left int64, right int64, mkey int64, mvalue Value_t, size int) {
	self.mx.Lock()
	left = self.left
	right = self.right
	mkey = self.median.Key
	mvalue = self.median.Value
	size = self.cc.Size()
	self.mx.Unlock()
	return
}
