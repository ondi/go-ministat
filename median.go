//
//
//

package ministat

import (
	"time"

	"github.com/ondi/go-cache"
)

type Compare_t[T any] func(a T, b T) int

type Mapped_t[T any] struct {
	Ts   time.Time
	Data T
}

type Median_t[T any] struct {
	cx     *cache.Cache_t[int, Mapped_t[T]]
	median *cache.Value_t[int, Mapped_t[T]]
	ttl    time.Duration
	seq    int
	limit  int
	left   int
	right  int
}

func NewMedian[T any](limit int, ttl time.Duration) (self *Median_t[T]) {
	self = &Median_t[T]{
		cx:    cache.New[int, Mapped_t[T]](),
		ttl:   ttl,
		limit: limit,
		right: -1,
	}
	self.median = self.cx.End()
	return
}

func (self *Median_t[T]) Add(ts time.Time, data T, cmp Compare_t[T]) (T, int) {
	self.Evict(ts, cmp)
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	it, inserted := self.cx.CreateBack(
		self.seq,
		func(p *Mapped_t[T]) {
			p.Ts = ts
			p.Data = data
		},
		func(p *Mapped_t[T]) {},
	)
	if inserted {
		if self.cx.Size() == 1 {
			self.median = it
			self.right++
		} else if cmp(it.Value.Data, self.median.Value.Data) > 0 {
			self.right++
		} else {
			self.left++
		}
	} else {
		if it == self.median {
			self.median = self.median.Next()
			self.left++
			self.right--
		}
		// если новое значения элемента остаётся в той же половине списка,
		// коррекция указалетей left и right не требуется.
		if cmp(data, self.median.Value.Data) > 0 {
			if cmp(it.Value.Data, self.median.Value.Data) < 0 {
				self.left--
				self.right++
			}
		} else {
			if cmp(it.Value.Data, self.median.Value.Data) >= 0 {
				self.left++
				self.right--
			}
		}
		it.Value.Ts = ts
		it.Value.Data = data
	}
	// sort value
	at := self.cx.Front()
	for ; at != self.cx.End(); at = at.Next() {
		if cmp(it.Value.Data, at.Value.Data) <= 0 && it != at {
			break
		}
	}
	cache.CutList(it)
	cache.SetPrev(it, at)
	self.move_median()
	return self.median.Value.Data, self.cx.Size()
}

func (self *Median_t[T]) Evict(ts time.Time, cmp Compare_t[T]) int {
	begin := self.begin()
	for self.cx.Size() > 0 {
		it, _ := self.cx.Find(begin)
		if ts.Sub(it.Value.Ts) < self.ttl {
			return self.cx.Size()
		}
		self.remove(it, cmp)
		begin++
		if begin >= self.limit {
			begin = 0
		}
	}
	return 0
}

func (self *Median_t[T]) Median(ts time.Time, cmp Compare_t[T]) (median T, size int) {
	size, median = self.Evict(ts, cmp), self.median.Value.Data
	return
}

func (self *Median_t[T]) begin() (begin int) {
	if self.seq < self.cx.Size() {
		begin = self.limit - (self.cx.Size() - self.seq) + 1
	} else {
		begin = self.seq - self.cx.Size() + 1
	}
	if begin >= self.limit {
		begin = 0
	}
	return
}

func (self *Median_t[T]) remove(it *cache.Value_t[int, Mapped_t[T]], cmp Compare_t[T]) {
	if temp := cmp(it.Value.Data, self.median.Value.Data); temp < 0 {
		self.cx.Remove(it.Key)
		self.left--
	} else if temp > 0 {
		self.cx.Remove(it.Key)
		self.right--
	} else {
		if it != self.median {
			cache.Swap(it, self.median)
		}
		self.cx.Remove(it.Key)
		self.median = it.Next()
		self.right--
	}
	self.move_median()
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

func (self *Median_t[T]) range_test(ts time.Time, cmp Compare_t[T], f func(key int, value Mapped_t[T]) bool) {
	self.Evict(ts, cmp)
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if f(it.Key, it.Value) == false {
			return
		}
	}
}
