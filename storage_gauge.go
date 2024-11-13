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
	Name   string `json:"name"`
	Status string `json:"status"`
	Value  T      `json:"value"`
}

func (self Gauge_t[T]) GetName() string {
	return self.Name
}

func (self Gauge_t[T]) GetStatus() string {
	return self.Status
}

func (self Gauge_t[T]) GetValueInt64() int64 {
	return (int64)(self.Value)
}

func (self Gauge_t[T]) GetValueString() string {
	return fmt.Sprintf("%v", self.Value)
}

func (self Gauge_t[T]) String() string {
	if len(self.Status) > 0 {
		return fmt.Sprintf("{%s:%v %q}", self.Name, self.Value, self.Status)
	}
	return fmt.Sprintf("{%s:%v}", self.Name, self.Value)
}
