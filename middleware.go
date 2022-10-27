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

type PageName_t func(*http.Request) string
type LogCtx_t func(ctx context.Context, format string, args ...interface{})
type GetErr_t func(ctx context.Context, sb *strings.Builder) *strings.Builder

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

type Middleware_t struct {
	storage   *Storage_t
	median    *StorageMedian_t[time.Duration]
	ok        http.Handler
	not_ok    http.Handler
	page_name PageName_t
	log       LogCtx_t
	errors    GetErr_t
	views     Views
}

func NewMiddleware(storage *Storage_t, ok http.Handler, not_ok http.Handler, errors GetErr_t, log LogCtx_t, views Views, page_name PageName_t) *Middleware_t {
	return &Middleware_t{
		storage:   storage,
		median:    NewStorageMedian[time.Duration](128, 32),
		ok:        ok,
		not_ok:    not_ok,
		page_name: page_name,
		log:       log,
		errors:    errors,
		views:     views,
	}
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}

	p, sampling, state := self.storage.MetricBegin(page, start)

	var err error
	if sampling > 0 {
		if err = self.views.MinistatBefore(r.Context(), page); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
	if sampling == 0 || state != 0 {
		self.not_ok.ServeHTTP(&writer, r)
	} else {
		self.ok.ServeHTTP(&writer, r)
	}
	if sampling > 0 {
		if err = self.views.MinistatAfter(r.Context(), page); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}

	end := time.Now()
	diff := end.Sub(start)
	median, _ := self.median.Add(page, diff, CmpDuration)
	if self.storage.MetricEnd(p, diff, 1, writer.status_code) > 0 {
		var sb strings.Builder
		if err = self.views.MinistatDuration(r.Context(), page, diff, median, 1, writer.status_code, self.errors(r.Context(), &sb).String()); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
}
