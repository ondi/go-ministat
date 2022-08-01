//
//
//

package ministat

import (
	"net/http"
	"sync"
	"time"

	"github.com/ondi/go-log"
)

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

type Middleware_t struct {
	storage        *Storage_t
	next           http.Handler
	views          Views
	page_name      func(*http.Request) string
	state_mx       sync.Mutex
	state_ts       time.Time
	state_prev     int64
	state_next     int64
	state_limit    int64
	state_duration time.Duration
}

type Options func(self *Middleware_t)

func OnlineLimit(limit int64, duration time.Duration) Options {
	return func(self *Middleware_t) {
		self.state_limit = limit
		self.state_duration = duration
	}
}

func PageName(f func(*http.Request) string) Options {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, next http.Handler, views Views, opts ...Options) (self *Middleware_t) {
	self = &Middleware_t{
		storage:     storage,
		next:        next,
		views:       views,
		page_name:   GetPageName,
		state_limit: 1<<63 - 1,
		state_prev:  1,
		state_next:  1,
	}
	for _, v := range opts {
		v(self)
	}
	return
}

func (self *Middleware_t) __set_state(ts time.Time, state int64) int64 {
	if self.state_next != state {
		self.state_next = state
		self.state_ts = ts
	}
	if self.state_prev != state && ts.Sub(self.state_ts) >= self.state_duration {
		self.state_prev = state
	}
	return self.state_prev
}

func (self *Middleware_t) CheckState(ts time.Time, online int64) (state int64) {
	self.state_mx.Lock()
	if online >= self.state_limit {
		state = self.__set_state(ts, 2)
	} else {
		state = self.__set_state(ts, 1)
	}
	self.state_mx.Unlock()
	return
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}

	p, c := self.storage.MetricBegin(page, start)

	if c.Sampling > 0 {
		self.views.MinistatBefore(r.Context(), page)
	}
	if self.CheckState(start, c.Online) == 2 {
		log.WarnCtx(r.Context(), "TOO MANY REQUESTS: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	} else {
		self.next.ServeHTTP(&writer, r)
	}
	if c.Sampling > 0 {
		self.views.MinistatAfter(r.Context(), page)
	}

	diff := time.Since(start)
	if self.storage.MetricEnd(p, diff, 1, writer.status_code).Sampling > 0 {
		self.views.MinistatDuration(r.Context(), page, diff, 1, writer.status_code)
	}
}
