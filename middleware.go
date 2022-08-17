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

func PageName(r *http.Request) (res string) {
	return r.URL.Path
}

func NoErrors(ctx context.Context, sb *strings.Builder) *strings.Builder {
	return sb
}

func NoLog(ctx context.Context, format string, args ...interface{}) {}

type PageName_t func(*http.Request) string

type Middleware_t struct {
	storage   *Storage_t
	ok        http.Handler
	not_ok    http.Handler
	errors    GetErr_t
	log       LogCtx_t
	views     Views
	page_name PageName_t
}

func NewMiddleware(storage *Storage_t, ok http.Handler, not_ok http.Handler, errors GetErr_t, log LogCtx_t, views Views, page_name PageName_t) *Middleware_t {
	return &Middleware_t{
		storage:   storage,
		ok:        ok,
		not_ok:    not_ok,
		errors:    errors,
		log:       log,
		views:     views,
		page_name: page_name,
	}
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
		self.not_ok.ServeHTTP(&writer, r)
	} else {
		self.ok.ServeHTTP(&writer, r)
	}
	if c.Sampling > 0 {
		if err = self.views.MinistatAfter(r.Context(), page); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}

	diff := time.Since(start)
	if self.storage.MetricEnd(p, diff, 1, writer.status_code).Sampling > 0 {
		var sb strings.Builder
		if err = self.views.MinistatDuration(r.Context(), page, diff, 1, writer.status_code, self.errors(r.Context(), &sb).String()); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
}
