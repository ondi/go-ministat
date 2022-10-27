//
//
//

package ministat

import (
	"github.com/ondi/go-cache"
)

type Compare_t[T any] func(a T, b T) int

type Mapped_t[Measure_t any] struct {
	// Ts      time.Time
	Measure Measure_t
}

type Median_t[Measure_t any] struct {
	cc     *cache.Cache_t[int, Mapped_t[Measure_t]]
	median *cache.Value_t[int, Mapped_t[Measure_t]]
	seq    int
	limit  int
	left   int
	right  int
}

func NewMedian[Measure_t any](limit int) (self *Median_t[Measure_t]) {
	self = &Median_t[Measure_t]{
		cc:    cache.New[int, Mapped_t[Measure_t]](),
		limit: limit,
		right: -1,
	}
	self.median = self.cc.End()
	return
}

func (self *Median_t[Measure_t]) Add(measure Measure_t, cmp Compare_t[Measure_t]) (res Measure_t) {
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	var less_before bool
	it, inserted := self.cc.CreateBack(self.seq, func() Mapped_t[Measure_t] {
		return Mapped_t[Measure_t]{Measure: measure}
	})
	if inserted {
		if self.cc.Size() == 1 {
			self.median = it
		}
	} else {
		// определяем из какой половины списка перезаписываемый элемент
		// чтобы скорректировать число элементов слева и справа от медианы или оставить как есть
		if cmp(it.Value.Measure, self.median.Value.Measure) < 0 {
			less_before = true
		}
		if it == self.median {
			self.median = self.median.Next()
			less_before = true
			self.left++
			self.right--
		}
		it.Value.Measure = measure
	}
	median_passed := self.move_value(it, cmp)
	self.set_median(it, median_passed, inserted, less_before)
	res = self.median.Value.Measure
	return
}

func (self *Median_t[Measure_t]) remove(it *cache.Value_t[int, Mapped_t[Measure_t]], cmp Compare_t[Measure_t]) {
	// нет способа определить из какой половины элемент, кроме полного прохода по списку
}

func (self *Median_t[Measure_t]) move_value(it *cache.Value_t[int, Mapped_t[Measure_t]], cmp Compare_t[Measure_t]) (median_passed bool) {
	for at := self.cc.Front(); at != self.cc.End(); at = at.Next() {
		if cmp(it.Value.Measure, at.Value.Measure) <= 0 && it != at {
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

func (self *Median_t[Measure_t]) set_median(it *cache.Value_t[int, Mapped_t[Measure_t]], median_passed bool, inserted bool, less_before bool) {
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

func (self *Median_t[Measure_t]) Median() (res Measure_t) {
	return self.median.Value.Measure
}

func (self *Median_t[Measure_t]) Size() (res int) {
	return self.cc.Size()
}

func (self *Median_t[Measure_t]) Range(f func(key int, value Measure_t) bool) {
	for it := self.cc.Front(); it != self.cc.End(); it = it.Next() {
		if f(it.Key, it.Value.Measure) == false {
			return
		}
	}
}

func (self *Median_t[Measure_t]) DebugLR() (left int, right int, mkey int, mvalue Measure_t, size int) {
	left = self.left
	right = self.right
	mkey = self.median.Key
	mvalue = self.median.Value.Measure
	size = self.cc.Size()
	return
}
