//
//
//

package ministat

import (
	"time"

	"github.com/ondi/go-cache"
)

// value will be compared with greater and less operators
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type MedianMapped_t[T Number] struct {
	Ts   time.Time
	Data T
}

type Median_t[T Number] struct {
	cx     *cache.Cache_t[int, MedianMapped_t[T]]
	median *cache.Value_t[int, MedianMapped_t[T]]
	sum    T
	ttl    time.Duration
	seq    int
	limit  int
	left   int
	right  int
}

func NewMedian[T Number](limit int, ttl time.Duration) (self *Median_t[T]) {
	self = &Median_t[T]{
		cx:    cache.New[int, MedianMapped_t[T]](),
		ttl:   ttl,
		limit: limit,
		right: -1,
	}
	self.median = self.cx.End()
	return
}

// med, avg, max, size
func (self *Median_t[T]) Add(ts time.Time, data T) (T, T, T, int) {
	self.Evict(ts)
	it, inserted := self.cx.CreateBack(
		self.seq,
		func(p *MedianMapped_t[T]) {
			p.Ts = ts.Add(self.ttl)
			p.Data = data
		},
		func(p *MedianMapped_t[T]) {
			// do not overwrite value here it.Value.Data used below
		},
	)
	self.sum += data
	self.seq++
	if self.seq >= self.limit {
		self.seq = 0
	}
	if inserted {
		if self.cx.Size() == 1 {
			self.median = it
			self.right++
		} else if it.Value.Data > self.median.Value.Data {
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
		// если перезаписываемое значения остаётся в той же половине списка,
		// коррекция указалетей left и right не требуется.
		if data > self.median.Value.Data {
			if it.Value.Data < self.median.Value.Data {
				self.left--
				self.right++
			}
		} else {
			if it.Value.Data >= self.median.Value.Data {
				self.left++
				self.right--
			}
		}
		self.sum -= it.Value.Data
		it.Value.Ts = ts.Add(self.ttl)
		it.Value.Data = data
	}
	// insert value into sorted list
	at := self.median
	if it.Value.Data > self.median.Value.Data {
		for ; at != self.cx.End(); at = at.Next() {
			if it.Value.Data <= at.Value.Data && it != at {
				break
			}
		}
		cache.CutList(it)
		cache.SetPrev(it, at)
	} else {
		for ; at != self.cx.End(); at = at.Prev() {
			if it.Value.Data > at.Value.Data && it != at {
				break
			}
		}
		cache.CutList(it)
		cache.SetNext(it, at)
	}
	self.move_median()
	return self.median.Value.Data, self.sum / T(self.cx.Size()), self.cx.Back().Value.Data, self.cx.Size()
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

func (self *Median_t[T]) Evict(ts time.Time) int {
	begin := self.begin()
	for {
		if it, ok := self.cx.Find(begin); ok && ts.Before(it.Value.Ts) == false {
			self.sum -= it.Value.Data
			self.remove(it)
			begin++
			if begin >= self.limit {
				begin = 0
			}
		} else {
			return self.cx.Size()
		}
	}
	return 0
}

func (self *Median_t[T]) begin() (begin int) {
	if begin = self.seq - self.cx.Size(); begin < 0 {
		begin += self.limit
	}
	return
}

func (self *Median_t[T]) Value(ts time.Time) (med T, avg T, max T, size int) {
	if size = self.Evict(ts); size > 0 {
		avg = self.sum / T(size)
	}
	med = self.median.Value.Data
	max = self.cx.Back().Value.Data
	return
}

func (self *Median_t[T]) remove(it *cache.Value_t[int, MedianMapped_t[T]]) {
	if it.Value.Data < self.median.Value.Data {
		self.cx.Remove(it.Key)
		self.left--
	} else if it.Value.Data > self.median.Value.Data {
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

func (self *Median_t[T]) range_test(ts time.Time, f func(key int, value MedianMapped_t[T]) bool) {
	self.Evict(ts)
	for it := self.cx.Front(); it != self.cx.End(); it = it.Next() {
		if f(it.Key, it.Value) == false {
			return
		}
	}
}
