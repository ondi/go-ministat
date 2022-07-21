//
//
//

package ministat

import (
	"net/http"
	"time"

	"github.com/ondi/go-log"
)

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

type Middleware_t struct {
	storage   *Storage_t
	next      http.Handler
	views     Views
	page_name func(*http.Request) string
	limit     int64
}

type StatOption func(self *Middleware_t)

func LimitPerPage(limit int64) StatOption {
	return func(self *Middleware_t) {
		self.limit = limit
	}
}

func PageName(f func(*http.Request) string) StatOption {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, next http.Handler, views Views, opts ...StatOption) (self *Middleware_t) {
	self = &Middleware_t{
		storage: storage,
		next:    next,
		views:   views,
	}
	for _, v := range opts {
		v(self)
	}
	if self.page_name == nil {
		self.page_name = GetPageName
	}
	if self.limit <= 0 {
		self.limit = 1<<63 - 1
	}
	return
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}

	counter, current := self.storage.MetricBegin(page, start)

	if current.Ref > 0 {
		self.views.MinistatBefore(r.Context(), page)
	}
	if current.Online >= self.limit {
		log.WarnCtx(r.Context(), "TOO MANY REQUESTS: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	} else {
		self.next.ServeHTTP(&writer, r)
	}
	if current.Ref > 0 {
		self.views.MinistatAfter(r.Context(), page)
	}

	diff := time.Since(start)
	if self.storage.MetricEnd(counter, diff, 1, writer.status_code).Ref > 0 {
		self.views.MinistatDuration(r.Context(), page, diff, 1, writer.status_code)
	}
}
