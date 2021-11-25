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

var NOONLINE = NoOnline_t{}
var PAGENAME = PageName_t{}

type PageName interface {
	GetPageName(r *http.Request) (res string)
}

type PageName_t struct{}

func (PageName_t) GetPageName(r *http.Request) (res string) {
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
	MinistatDuration(r *http.Request, name string, status int, diff time.Duration)
}

type NoOnline_t struct{}

func (NoOnline_t) MinistatContext(r *http.Request) *http.Request {
	return r
}

func (NoOnline_t) MinistatOnline(w http.ResponseWriter, r *http.Request, count int64) bool {
	return true
}

func (NoOnline_t) MinistatDuration(r *http.Request, name string, status int, diff time.Duration) {
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
	truncate      time.Duration
	limit_backlog int
	limit_items   int
}

func NewStorage(limit_backlog int, limit_items int, truncate time.Duration) (self *Storage_t) {
	self = &Storage_t{
		cc:            cache.New(),
		truncate:      truncate,
		limit_backlog: limit_backlog,
		limit_items:   limit_items,
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

func (self *Storage_t) MetricEnd(counter *Counter_t, diff time.Duration, processed int, status_code int) {
	self.mx.Lock()
	counter.Online--
	counter.DurationSum += diff
	counter.Processed += processed
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

func (self *Storage_t) AddDuration(name string, start time.Time, diff time.Duration, processed int, status_code int) {
	self.MetricEnd(self.MetricBegin(name, start), diff, processed, status_code)
}

func (self *Storage_t) List(order cache.MyLess, limit int) (res []Stat_t) {
	self.mx.Lock()
	defer self.mx.Unlock()
	for it := self.cc.Back(); it != self.cc.End(); it = it.Prev() {
		if limit == 0 {
			return
		}
		limit--
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
	storage   *Storage_t
	next      http.Handler
	online    Online
	page_name PageName
}

func NewMiddleware(storage *Storage_t, next http.Handler, online Online, page_name PageName) (self *Middleware_t) {
	self = &Middleware_t{
		storage:   storage,
		next:      next,
		online:    online,
		page_name: page_name,
	}
	return
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	writer := StatusResponseWriter{w, http.StatusOK}

	name := self.page_name.GetPageName(r)
	counter := self.storage.MetricBegin(name, start)

	r = self.online.MinistatContext(r)

	if self.online.MinistatOnline(&writer, r, counter.Online) {
		self.next.ServeHTTP(&writer, r)
	}

	diff := time.Since(start)
	self.online.MinistatDuration(r, name, writer.status_code, diff)

	self.storage.MetricEnd(counter, diff, 1, writer.status_code)
}

func (self *Middleware_t) List(order cache.MyLess, limit int) (res []Stat_t) {
	return self.storage.List(order, limit)
}

type LessHits_t struct{}

func (LessHits_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationNum < b.Value.(*Counter_t).DurationNum
}

type LessProcessed_t struct{}

func (LessProcessed_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).Processed < b.Value.(*Counter_t).Processed
}

type LessDuration_t struct{}

func (LessDuration_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Value.(*Counter_t).DurationSum/a.Value.(*Counter_t).DurationNum < b.Value.(*Counter_t).DurationSum/b.Value.(*Counter_t).DurationNum
}

type LessName_t struct{}

func (LessName_t) Less(a *cache.Value_t, b *cache.Value_t) bool {
	return a.Key.(string) < b.Key.(string)
}
