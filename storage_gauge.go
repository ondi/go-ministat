//
//
//

package ministat

import (
	"fmt"
	"net/http"
	"time"
)

// [Key_t comparable] for storage
type Page_t struct {
	Entry string `json:"entry"` // shard
	Name  string `json:"name"`  // page
}

func GetPageName(r *http.Request) Page_t {
	return Page_t{
		Name: r.URL.Path,
	}
}

type _429_t struct {
	log  LogWrite_t
	ts   time.Time
	diff time.Duration
}

func New429(log LogWrite_t, diff time.Duration) http.Handler {
	self := &_429_t{
		log:  log,
		diff: diff,
	}
	return self
}

func (self *_429_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	if ts.Sub(self.ts) > self.diff {
		self.ts = ts
		self.log(r.Context(), "TOO MANY REQUESTS: %q", r.URL.Path)
	}
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

type Gauge_t[T ~int64 | ~float64] struct {
	Name  string `json:"name"`
	Level string `json:"level"`
	Tag   string `json:"tag"`
	Value T      `json:"value"`
}

func (self Gauge_t[T]) GetName() string {
	return self.Name
}

func (self Gauge_t[T]) GetLevel() string {
	return self.Level
}

func (self Gauge_t[T]) GetTag() string {
	return self.Tag
}

func (self Gauge_t[T]) GetValue() T {
	return self.Value
}

func (self Gauge_t[T]) GetValueInt64() int64 {
	return (int64)(self.Value)
}

func (self Gauge_t[T]) GetValueFloat64() float64 {
	return (float64)(self.Value)
}

func (self Gauge_t[T]) String() string {
	return fmt.Sprintf("{%s:%s:%s:%v}", self.Name, self.Level, self.Tag, self.Value)
}

type GaugeList_t[T ~int64 | ~float64] []Gauge_t[T]

func (self GaugeList_t[T]) Len() int {
	return len(self)
}

func (self GaugeList_t[T]) Less(i int, j int) bool {
	return self[i].Value < self[j].Value ||
		self[i].Value == self[j].Value && self[i].Level < self[j].Level ||
		self[i].Value == self[j].Value && self[i].Level == self[j].Level && self[i].Tag < self[j].Tag
}

func (self GaugeList_t[T]) Swap(i int, j int) {
	self[i], self[j] = self[j], self[i]
}
