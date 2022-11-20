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

func (self *Median_t[T]) Add(ts time.Time, data T, cmp Compare_t[T]) (median T, size int) {
	self.evict(ts, cmp)
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	var less_before bool
	it, inserted := self.cx.CreateBack(self.seq, func() Mapped_t[T] {
		return Mapped_t[T]{Ts: ts, Data: data}
	})
	if inserted {
		if self.cx.Size() == 1 {
			self.median = it
		}
	} else {
		// если перезаписываемый элемент и новый элемент находятся в одной и той же
		// половине списка от медианы коррекция указалетей left, right не требуется
		if it == self.median {
			less_before = true
			self.median = self.median.Next()
			self.left++
			self.right--
		} else {
			less_before = cmp(it.Value.Data, self.median.Value.Data) < 0
		}
		it.Value.Data = data
	}
	// move value
	at := self.cx.Front()
	for ; at != self.cx.End(); at = at.Next() {
		if cmp(it.Value.Data, at.Value.Data) <= 0 && it != at {
			break
		}
	}
	cache.CutList(it)
	cache.SetPrev(it, at)
	// median passed
	if cmp(it.Value.Data, self.median.Value.Data) > 0 || it == self.median {
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
	self.move_median()
	median, size = self.median.Value.Data, self.cx.Size()
	return
}

func (self *Median_t[T]) evict(ts time.Time, cmp Compare_t[T]) {
	var begin int
	if self.seq < self.cx.Size() {
		begin = self.limit - (self.cx.Size() - self.seq)
	} else {
		begin = self.seq - self.cx.Size()
	}
	var it *cache.Value_t[int, Mapped_t[T]]
	for self.cx.Size() > 0 {
		begin++
		if begin >= self.limit {
			begin = 0
		}
		it, _ = self.cx.Find(begin)
		if ts.Sub(it.Value.Ts) < self.ttl {
			return
		}
		self.remove(it, cmp)
	}
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

func (self *Median_t[T]) Median(ts time.Time, cmp Compare_t[T]) (median T, median_ts time.Time, size int) {
	self.evict(ts, cmp)
	median, median_ts, size = self.median.Value.Data, self.median.Value.Ts, self.cx.Size()
	return
}

func (self *Median_t[T]) Range(f func(key int, value T) bool) {
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if f(it.Key, it.Value.Data) == false {
			return
		}
	}
}

func (self *Median_t[T]) debug_state() (left int, right int, mkey int, mvalue T, size int) {
	left = self.left
	right = self.right
	mkey = self.median.Key
	mvalue = self.median.Value.Data
	size = self.cx.Size()
	return
}

func (self *Median_t[T]) debug_remove(key int, cmp Compare_t[T]) {
	if it, ok := self.cx.Find(key); ok {
		self.remove(it, cmp)
	}
}
