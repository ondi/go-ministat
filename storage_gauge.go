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

type Gauge_t[T ~int64] struct {
	Type   string `json:"type"`
	Result string `json:"result"`
	Value  T      `json:"value"`
}

func (self Gauge_t[T]) GetType() string {
	return self.Type
}

func (self Gauge_t[T]) GetResult() string {
	return self.Result
}

func (self Gauge_t[T]) GetValueInt64() int64 {
	return (int64)(self.Value)
}

func (self Gauge_t[T]) String() string {
	if len(self.Result) > 0 {
		return fmt.Sprintf("{%s:%v %q}", self.Type, self.Value, self.Result)
	}
	return fmt.Sprintf("{%s:%v}", self.Type, self.Value)
}
