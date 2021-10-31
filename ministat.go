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
	MinistatOnline(r *http.Request, count int64) (*http.Request, error)
	MinistatDuration(r *http.Request, status int, diff time.Duration)
	MinistatEvict(key interface{})
}

type NoOnline_t struct{}

func (NoOnline_t) MinistatOnline(r *http.Request, count int64) (*http.Request, error) {
	return r, nil
}

func (NoOnline_t) MinistatDuration(r *http.Request, status int, diff time.Duration) {}
func (NoOnline_t) MinistatEvict(key interface{})                                    {}

type Ministat_t struct {
	mx            sync.Mutex
	cc            *cache.Cache_t // key = ts.Truncate(self.truncate), value = *unique.Often_t
	limit_backlog int
	limit_items   int
	truncate      time.Duration
	begin         time.Time
	online        Online
	next          http.Handler
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

func New(limit_backlog int, limit_items int, truncate time.Duration, next http.Handler, online Online) (self *Ministat_t) {
	self = &Ministat_t{
		cc:            cache.New(),
		limit_backlog: limit_backlog,
		limit_items:   limit_items,
		truncate:      truncate,
		begin:         time.Now(),
		online:        online,
		next:          next,
	}
	return
}

func (self *Ministat_t) MetricBegin(name string, start time.Time, processed int) (counter *Counter_t) {
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
		self.online.MinistatEvict(key)
	}
	counter, _ = it.Value.(*unique.Often_t).Add(name, func() unique.Counter { return &Counter_t{} }).(*Counter_t)
	counter.Online++
	if counter.Online > counter.OnlineMax {
		counter.OnlineMax = counter.Online
	}
	counter.Processed += processed
	counter.DurationNum++
	self.mx.Unlock()
	return
}

func (self *Ministat_t) MetricEnd(counter *Counter_t, diff time.Duration, status_code int) {
	self.mx.Lock()
	counter.Online--
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

func (self *Ministat_t) AddDuration(name string, processed int, start time.Time, diff time.Duration, status_code int) {
	counter := self.MetricBegin(name, start, processed)
	self.MetricEnd(counter, diff, status_code)
}

func (self *Ministat_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	writer := StatusResponseWriter{w, http.StatusOK}

	counter := self.MetricBegin(GETURL(r), start, 1)

	var diff time.Duration
	req, err := self.online.MinistatOnline(r, counter.Online)
	if err != nil {
		http.Error(&writer, err.Error(), http.StatusTooManyRequests)
	} else {
		self.next.ServeHTTP(&writer, req)
	}
	diff = time.Since(start)
	self.online.MinistatDuration(req, writer.status_code, diff)

	self.MetricEnd(counter, diff, writer.status_code)
}

func (self *Ministat_t) List(order cache.MyLess) (res []Stat_t) {
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

type LessHits_t struct{}

func (LessHits_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationNum < b.Value.(*Counter_t).DurationNum
}

type LessDuration_t struct{}

func (LessDuration_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationMax < b.Value.(*Counter_t).DurationMax
}
