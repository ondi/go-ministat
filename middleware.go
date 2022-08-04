//
//
//

package ministat

import (
	"net/http"
	"strings"
	"time"

	"github.com/ondi/go-log"
)

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

func TrimValue(s string, out *strings.Builder) *strings.Builder {
	if len(s) > 255 {
		s = s[:255]
	}
	for _, r := range s {
		if r >= 0x20 && r <= 0x7e {
			out.WriteRune(r)
		}
	}
	return out
}

type Middleware_t struct {
	storage   *Storage_t
	next      http.Handler
	views     Views
	page_name func(*http.Request) string
}

type MiddlewareOptions func(self *Middleware_t)

func MiddlewarePageName(f func(*http.Request) string) MiddlewareOptions {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, next http.Handler, views Views, opts ...MiddlewareOptions) (self *Middleware_t) {
	self = &Middleware_t{
		storage:   storage,
		next:      next,
		views:     views,
		page_name: GetPageName,
	}
	for _, v := range opts {
		v(self)
	}
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
	if c.Sampling == 0 || c.State == 1 {
		if c.StateTs.Equal(start) {
			log.WarnCtx(r.Context(), "TOO MANY REQUESTS: %v %v", c.Online, page)
		}
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	} else {
		self.next.ServeHTTP(&writer, r)
	}
	if c.Sampling > 0 {
		self.views.MinistatAfter(r.Context(), page)
	}

	diff := time.Since(start)
	var sb strings.Builder
	if v := log.ContextGet(r.Context()); v != nil {
		TrimValue(strings.Join(v.Values(), ","), &sb)
	}
	if self.storage.MetricEnd(p, diff, 1, writer.status_code).Sampling > 0 {
		self.views.MinistatDuration(r.Context(), page, diff, 1, writer.status_code, sb.String())
	}
}
