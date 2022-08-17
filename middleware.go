//
//
//

package ministat

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type _429_t struct {
	log  LogCtx_t
	ts   time.Time
	diff time.Duration
}

func New429(log LogCtx_t, diff time.Duration) http.Handler {
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

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

func NoErrors(ctx context.Context, sb *strings.Builder) *strings.Builder {
	return sb
}

func NoLog(ctx context.Context, format string, args ...interface{}) {}

type Middleware_t struct {
	storage   *Storage_t
	ok        http.Handler
	err       http.Handler
	errors    GetErr_t
	log       LogCtx_t
	views     Views
	page_name func(*http.Request) string
}

type MiddlewareOptions func(self *Middleware_t)

func MiddlewarePageName(f func(*http.Request) string) MiddlewareOptions {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, ok http.Handler, err http.Handler, errors GetErr_t, log LogCtx_t, views Views, opts ...MiddlewareOptions) (self *Middleware_t) {
	self = &Middleware_t{
		storage:   storage,
		ok:        ok,
		err:       err,
		errors:    errors,
		log:       log,
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

	var err error
	if c.Sampling > 0 {
		if err = self.views.MinistatBefore(r.Context(), page); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
	if c.Sampling == 0 || c.State != 0 {
		self.err.ServeHTTP(&writer, r)
	} else {
		self.ok.ServeHTTP(&writer, r)
	}
	if c.Sampling > 0 {
		if err = self.views.MinistatAfter(r.Context(), page); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}

	end := time.Now()
	if self.storage.MetricEnd(p, end, 1, writer.status_code).Sampling > 0 {
		var sb strings.Builder
		if err = self.views.MinistatDuration(r.Context(), page, end.Sub(start), 1, writer.status_code, self.errors(r.Context(), &sb).String()); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
}
