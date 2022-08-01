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
	storage         *Storage_t
	next            http.Handler
	views           Views
	page_name       func(*http.Request) string
	online_mx       sync.Mutex
	online_ts       time.Time
	online_state    int64
	online_limit    int64
	online_duration time.Duration
}

type Options func(self *Middleware_t)

func OnlineLimit(limit int64, duration time.Duration) Options {
	return func(self *Middleware_t) {
		self.online_limit = limit
		self.online_duration = duration
	}
}

func PageName(f func(*http.Request) string) Options {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, next http.Handler, views Views, opts ...Options) (self *Middleware_t) {
	self = &Middleware_t{
		storage:      storage,
		next:         next,
		views:        views,
		page_name:    GetPageName,
		online_limit: 1<<63 - 1,
		online_state: 1,
	}
	for _, v := range opts {
		v(self)
	}
	return
}

func (self *Middleware_t) CheckState(ts time.Time, online int64) (state int64, diff time.Duration) {
	self.online_mx.Lock()
	if online >= self.online_limit {
		if self.online_state == 2 {
			diff = ts.Sub(self.online_ts)
		} else {
			self.online_state = 2
			self.online_ts = ts
		}
	} else {
		if self.online_state == 1 {
			diff = ts.Sub(self.online_ts)
		} else {
			self.online_state = 1
			self.online_ts = ts
		}
	}
	state = self.online_state
	self.online_mx.Unlock()
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
	if state, duration := self.CheckState(start, c.Online); duration < self.online_duration || state == 2 {
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
