//
//
//

package ministat

import (
	"github.com/ondi/go-cache"
)

type Compare_t[T any] func(a T, b T) int

type Mapped_t[T any] struct {
	// Ts      time.Time
	Data T
}

type Median_t[T any] struct {
	cc     *cache.Cache_t[int, Mapped_t[T]]
	median *cache.Value_t[int, Mapped_t[T]]
	seq    int
	limit  int
	left   int
	right  int
}

func NewMedian[T any](limit int) (self *Median_t[T]) {
	self = &Median_t[T]{
		cc:    cache.New[int, Mapped_t[T]](),
		limit: limit,
	}
	self.median = self.cc.End()
	return
}

func (self *Median_t[T]) Add(data T, cmp Compare_t[T]) (res T) {
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	var less_before bool
	it, inserted := self.cc.CreateBack(self.seq, func() Mapped_t[T] {
		return Mapped_t[T]{Data: data}
	})
	if inserted {
		if self.cc.Size() == 1 {
			self.median = it
			res = data
			return
		}
	} else {
		// определяем из какой половины списка перезаписываемый элемент
		// чтобы скорректировать число элементов слева и справа от медианы или оставить как есть
		if cmp(it.Value.Data, self.median.Value.Data) < 0 {
			less_before = true
		} else if it == self.median {
			less_before = true
			self.median = self.median.Next()
			self.left++
			self.right--
		}
		it.Value.Data = data
	}
	median_passed := self.move_value(it, cmp)
	self.move_pointers(median_passed, inserted, less_before)
	self.move_median()
	res = self.median.Value.Data
	return
}

func (self *Median_t[T]) Remove(key int, cmp Compare_t[T]) {
	it, ok := self.cc.Find(key)
	if !ok {
		return
	}
	self.remove(it, cmp)
}

func (self *Median_t[T]) remove(it *cache.Value_t[int, Mapped_t[T]], cmp Compare_t[T]) {
	self.cc.Remove(it.Key)
	if self.cc.Size() == 0 {
		self.median = self.cc.End()
		self.left = 0
		self.right = 0
	} else if cmp(it.Value.Data, self.median.Value.Data) < 0 {
		self.left--
	} else if it == self.median {
		if self.median.Next() != self.cc.End() {
			self.median = self.median.Next()
			self.right--
		} else {
			self.median = self.median.Prev()
			self.left--
		}
	} else {
		self.right--
	}
	self.move_median()
}

func (self *Median_t[T]) move_value(it *cache.Value_t[int, Mapped_t[T]], cmp Compare_t[T]) (median_passed bool) {
	for at := self.cc.Front(); at != self.cc.End(); at = at.Next() {
		if cmp(it.Value.Data, at.Value.Data) <= 0 && it != at {
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

func (self *Median_t[T]) move_pointers(median_passed bool, inserted bool, less_before bool) {
	if median_passed {
		if inserted {
			self.right++
		} else if less_before {
			self.left--
			self.right++
		}
	} else {
		if inserted {
			self.left++
		} else if less_before == false {
			self.left++
			self.right--
		}
	}
}

func (self *Median_t[T]) move_median() {
	if self.right < self.left-1 {
		self.median = self.median.Prev()
		self.left--
		self.right++
	} else if self.left < self.right-1 {
		self.median = self.median.Next()
		self.left++
		self.right--
	}
}

func (self *Median_t[T]) Median() (res T) {
	return self.median.Value.Data
}

func (self *Median_t[T]) Size() (res int) {
	return self.cc.Size()
}

func (self *Median_t[T]) Range(f func(key int, value T) bool) {
	for it := self.cc.Front(); it != self.cc.End(); it = it.Next() {
		if f(it.Key, it.Value.Data) == false {
			return
		}
	}
}

func (self *Median_t[T]) DebugLR() (left int, right int, mkey int, mvalue T, size int) {
	left = self.left
	right = self.right
	mkey = self.median.Key
	mvalue = self.median.Value.Data
	size = self.cc.Size()
	return
}
