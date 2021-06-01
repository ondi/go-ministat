//
//
//

package ministat

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ondi/go-cache"
	"github.com/ondi/go-unique"
)

type Counter_t struct {
	Count      int64
	Online     int64
	OnlineMax  int64
	RequestMax time.Duration
	RequestSum time.Duration
	Status200  int64
	Status400  int64
	Status500  int64
	Status000  int64
}

func (self *Counter_t) CounterAdd(a int64) int64 {
	self.Count += a
	return self.Count
}

type Route_t struct {
	Name    string
	Counter Counter_t
}

type Stat_t struct {
	Ts     time.Time
	Routes []Route_t
}

type Online_t func(string, int64, time.Duration)

type Ministat_t struct {
	mx            sync.Mutex
	cc            *cache.Cache_t // key = ts.Truncate(self.truncate), value = *unique.Often_t
	limit_backlog int
	limit_items   int
	truncate      time.Duration
	online        Online_t
	next          http.Handler
}

func NoOnline(string, int64, time.Duration) {}

type StatusResponseWriter struct {
	http.ResponseWriter
	status_code int
}

func (self *StatusResponseWriter) WriteHeader(code int) {
	self.status_code = code
	self.ResponseWriter.WriteHeader(code)
}

func (self *StatusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := self.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("not a http.Hijacker")
}

func New(limit_backlog int, limit_items int, truncate time.Duration, next http.Handler, online Online_t) (self *Ministat_t) {
	self = &Ministat_t{
		cc:            cache.New(),
		limit_backlog: limit_backlog,
		limit_items:   limit_items,
		truncate:      truncate,
		online:        online,
		next:          next,
	}
	return
}

func (self *Ministat_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	writer := &StatusResponseWriter{w, http.StatusOK}

	self.mx.Lock()
	it, _ := self.cc.CreateBack(
		start.Truncate(self.truncate),
		func() interface{} {
			return unique.NewOften(self.limit_items)
		},
	)
	if self.cc.Size() > self.limit_backlog {
		self.cc.Remove(self.cc.Front().Key)
	}
	counter, _ := it.Value.(*unique.Often_t).Add(r.URL.Path, func() unique.Counter { return &Counter_t{} }).(*Counter_t)
	counter.Online++
	if counter.Online > counter.OnlineMax {
		counter.OnlineMax = counter.Online
	}
	self.mx.Unlock()

	self.next.ServeHTTP(writer, r)
	diff := time.Since(start)
	self.online(r.URL.Path, counter.Online, diff)

	self.mx.Lock()
	if diff > counter.RequestMax {
		counter.RequestMax = diff
	}
	counter.Online--
	counter.RequestSum += diff
	switch {
	case writer.status_code >= 200 && writer.status_code < 300:
		counter.Status200++
	case writer.status_code >= 400 && writer.status_code < 500:
		counter.Status400++
	case writer.status_code >= 500:
		counter.Status500++
	default:
		counter.Status000++
	}
	self.mx.Unlock()
}

func (self *Ministat_t) List(order int64) (res []Stat_t) {
	self.mx.Lock()
	defer self.mx.Unlock()
	for it := self.cc.Back(); it != self.cc.End(); it = it.Prev() {
		temp := Stat_t{
			Ts: it.Key.(time.Time),
		}
		switch order {
		case 1:
			it.Value.(*unique.Often_t).Range(
				LessRequest_t{},
				func(key interface{}, value unique.Counter) bool {
					temp.Routes = append(temp.Routes, Route_t{
						Name:    key.(string),
						Counter: *value.(*Counter_t),
					})
					return true
				},
			)
		default:
			it.Value.(*unique.Often_t).Range(
				unique.Less_t{},
				func(key interface{}, value unique.Counter) bool {
					temp.Routes = append(temp.Routes, Route_t{
						Name:    key.(string),
						Counter: *value.(*Counter_t),
					})
					return true
				},
			)
		}
		res = append(res, temp)
	}
	return
}

type LessRequest_t struct{}

func (LessRequest_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).RequestMax < b.Value.(*Counter_t).RequestMax
}
