//
//
//

package ministat

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Key_t for storage
type Page_t struct {
	Name  string // page name
	Entry string // for shards or something else
}

type Views[Key_t comparable] interface {
	HitBegin(ctx context.Context, page Key_t) (err error)
	HitEnd(ctx context.Context, page Key_t, median time.Duration, median_size int, processed int64, status string, errors string) (err error)
	HitReset(ctx context.Context, page Key_t, median time.Duration, median_size int) (err error)
}

type PageName_t[Key_t comparable] func(*http.Request) Key_t
type LogCtx_t func(ctx context.Context, format string, args ...interface{})
type GetErr_t func(ctx context.Context) []string

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

func GetPageName(r *http.Request) Page_t {
	return Page_t{
		Name:  r.URL.Path,
		Entry: "",
	}
}

func NoErrors(ctx context.Context, sb *strings.Builder) *strings.Builder {
	return sb
}

func NoLog(ctx context.Context, format string, args ...interface{}) {}

func CountErrors(status_code int) int64 {
	if status_code >= 400 {
		return 1
	}
	return 0
}

type Middleware_t[Key_t comparable] struct {
	storage       *Storage_t[Key_t]
	ok            http.Handler
	not_ok        http.Handler
	page_name     PageName_t[Key_t]
	log           LogCtx_t
	errors        GetErr_t
	views         Views[Key_t]
	pending_limit int64
}

func NewMiddleware[Key_t comparable](storage *Storage_t[Key_t], ok http.Handler, not_ok http.Handler, errors GetErr_t, log LogCtx_t, views Views[Key_t], page_name PageName_t[Key_t], pending_limit int64) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:       storage,
		ok:            ok,
		not_ok:        not_ok,
		page_name:     page_name,
		pending_limit: pending_limit,
		log:           log,
		errors:        errors,
		views:         views,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending := self.storage.HitBegin(page, ts)
	defer self.serve_done(r.Context(), counter, page, ts, &writer)
	err := self.views.HitBegin(r.Context(), page)
	if err != nil {
		self.log(r.Context(), "MINISTAT: %v %q", err, page)
	}
	if sampling > 0 && pending <= self.pending_limit {
		self.ok.ServeHTTP(&writer, r)
	} else {
		self.not_ok.ServeHTTP(&writer, r)
	}
}

func (self *Middleware_t[Key_t]) serve_done(ctx context.Context, counter *Counter_t, name Key_t, start time.Time, writer *ResponseWriter_t) {
	median, size := self.storage.HitEnd(counter, name, start, time.Now(), 1, CountErrors(writer.status_code))
	var errors string
	if temp := self.errors(ctx); len(temp) > 0 {
		if ix := strings.Index(temp[0], " "); ix > -1 {
			errors = temp[0][:ix]
		} else {
			errors = temp[0]
		}
	}
	err := self.views.HitEnd(ctx, name, median, size, 1, strconv.FormatInt(int64(writer.status_code), 10), errors)
	if err != nil {
		self.log(ctx, "MINISTAT: %v %q", err, name)
	}
}
