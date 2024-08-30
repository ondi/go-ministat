//
//
//

package ministat

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Key_t for storage
type Page_t struct {
	Entry string // shard
	Name  string // page name
}

type Gauge_t struct {
	Type   string
	Result string
	Value  int64
}

type Views[Key_t comparable] interface {
	HitCurrent(page Key_t, g []Gauge_t) (err error)
}

type GetPage_t[Key_t comparable] func(*http.Request) Key_t
type LogWrite_t func(ctx context.Context, format string, args ...interface{})
type LogRead_t func(ctx context.Context, f func(ts time.Time, file string, line int, level_name string, level_id int64, format string, args ...any) bool)

type _429_t struct {
	log  LogWrite_t
	ts   time.Time
	diff time.Duration
}

func New429(log LogWrite_t, diff time.Duration) http.Handler {
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
	next_passed   http.Handler
	next_failed   http.Handler
	page_name     GetPage_t[Key_t]
	log_read      LogRead_t
	views         Views[Key_t]
	pending_limit int64
}

func NewMiddleware[Key_t comparable](storage *Storage_t[Key_t], next_passed http.Handler, next_failed http.Handler, log_read LogRead_t, views Views[Key_t], page_name GetPage_t[Key_t], pending_limit int64) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:       storage,
		next_passed:   next_passed,
		next_failed:   next_failed,
		page_name:     page_name,
		pending_limit: pending_limit,
		log_read:      log_read,
		views:         views,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending, _ := self.storage.HitBegin(page, ts)
	defer self.serve_done(r.Context(), counter, page, ts, &writer)
	if sampling > 0 && pending <= self.pending_limit {
		self.next_passed.ServeHTTP(&writer, r)
	} else {
		self.next_failed.ServeHTTP(&writer, r)
	}
}

func (self *Middleware_t[Key_t]) serve_done(ctx context.Context, counter *Counter_t, name Key_t, start time.Time, writer *ResponseWriter_t) {
	var errors string
	self.log_read(ctx, func(ts time.Time, file string, line int, level_name string, level_id int64, format string, args ...any) bool {
		if level_id < 3 {
			return true
		}
		errors = FirstWords(fmt.Sprintf(format, args...), 3)
		return false
	})
	self.storage.HitEnd(counter, start, time.Now(), 1, strconv.FormatInt(int64(writer.status_code), 10), errors)
}

func FirstWords(in string, count int) string {
	for i, v := range in {
		if unicode.IsSpace(v) {
			count--
			if count <= 0 {
				return in[0:i]
			}
		}
	}
	return in
}
