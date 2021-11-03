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

var GETURL = func(r *http.Request) string {
	return r.URL.Path
}

type Counter_t struct {
	count       int64 // reservoir sampling
	Online      int64
	OnlineMax   int64
	Processed   int
	DurationNum time.Duration
	DurationSum time.Duration
	DurationMax time.Duration
	Status200   int64
	Status400   int64
	Status500   int64
	Status000   int64
}

func (self *Counter_t) CounterAdd(a int64) {
	self.count += a
}

func (self *Counter_t) CounterGet() int64 {
	return self.count
}

type Route_t struct {
	Name    string
	Counter Counter_t
}

type Stat_t struct {
	Ts     time.Time
	Routes []Route_t
}

type Online interface {
	MinistatContext(r *http.Request) *http.Request
	MinistatOnline(w http.ResponseWriter, r *http.Request, count int64) bool
	MinistatDuration(r *http.Request, status int, diff time.Duration)
}

type NoOnline_t struct{}

func (NoOnline_t) MinistatContext(r *http.Request) *http.Request {
	return r
}

func (NoOnline_t) MinistatOnline(w http.ResponseWriter, r *http.Request, count int64) bool {
	return true
}

func (NoOnline_t) MinistatDuration(r *http.Request, status int, diff time.Duration) {
	return
}

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

type Storage_t struct {
	mx            sync.Mutex
	cc            *cache.Cache_t // key = ts.Truncate(self.truncate), value = *unique.Often_t
	limit_backlog int
	limit_items   int
	truncate      time.Duration
}

func NewStorage(limit_backlog int, limit_items int, truncate time.Duration) (self *Storage_t) {
	self = &Storage_t{
		cc:            cache.New(),
		limit_backlog: limit_backlog,
		limit_items:   limit_items,
		truncate:      truncate,
	}
	return
}

func (self *Storage_t) MetricBegin(name string, start time.Time) (counter *Counter_t) {
	self.mx.Lock()
	it, _ := self.cc.CreateBack(
		start.Truncate(self.truncate),
		func() interface{} {
			return unique.NewOften(self.limit_items)
		},
	)
	if self.cc.Size() > self.limit_backlog {
		key := self.cc.Front().Key
		self.cc.Remove(key)
	}
	counter, _ = it.Value.(*unique.Often_t).Add(name, func() unique.Counter { return &Counter_t{} }).(*Counter_t)
	counter.Online++
	if counter.Online > counter.OnlineMax {
		counter.OnlineMax = counter.Online
	}
	counter.DurationNum++
	self.mx.Unlock()
	return
}

func (self *Storage_t) MetricEnd(counter *Counter_t, processed int, diff time.Duration, status_code int) {
	self.mx.Lock()
	counter.Online--
	counter.Processed += processed
	counter.DurationSum += diff
	if diff > counter.DurationMax {
		counter.DurationMax = diff
	}
	switch {
	case status_code >= 200 && status_code < 300:
		counter.Status200++
	case status_code >= 400 && status_code < 500:
		counter.Status400++
	case status_code >= 500:
		counter.Status500++
	default:
		counter.Status000++
	}
	self.mx.Unlock()
}

func (self *Storage_t) AddDuration(name string, processed int, start time.Time, diff time.Duration, status_code int) {
	self.MetricEnd(self.MetricBegin(name, start), processed, diff, status_code)
}

func (self *Storage_t) List(order cache.MyLess) (res []Stat_t) {
	self.mx.Lock()
	defer self.mx.Unlock()
	for it := self.cc.Back(); it != self.cc.End(); it = it.Prev() {
		temp := Stat_t{
			Ts: it.Key.(time.Time),
		}
		it.Value.(*unique.Often_t).Range(
			order,
			func(key interface{}, value unique.Counter) bool {
				temp.Routes = append(temp.Routes, Route_t{
					Name:    key.(string),
					Counter: *value.(*Counter_t),
				})
				return true
			},
		)
		res = append(res, temp)
	}
	return
}

type Middleware_t struct {
	storage *Storage_t
	next    http.Handler
	online  Online
}

func NewMiddleware(storage *Storage_t, next http.Handler, online Online) (self *Middleware_t) {
	self = &Middleware_t{
		storage: storage,
		next:    next,
		online:  online,
	}
	return
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	writer := StatusResponseWriter{w, http.StatusOK}

	counter := self.storage.MetricBegin(GETURL(r), start)

	r = self.online.MinistatContext(r)

	if self.online.MinistatOnline(&writer, r, counter.Online) {
		self.next.ServeHTTP(&writer, r)
	}

	diff := time.Since(start)
	self.online.MinistatDuration(r, writer.status_code, diff)

	self.storage.MetricEnd(counter, 1, diff, writer.status_code)
}

func (self *Middleware_t) List(order cache.MyLess) (res []Stat_t) {
	return self.storage.List(order)
}

type LessHits_t struct{}

func (LessHits_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationNum < b.Value.(*Counter_t).DurationNum
}

type LessDuration_t struct{}

func (LessDuration_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationMax < b.Value.(*Counter_t).DurationMax
}
