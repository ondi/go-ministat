//
//
//

package ministat

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type Gauge interface {
	GetType() string
	GetResult() string
	GetValueInt64() int64
	String() string
}

type Views[Key_t comparable] interface {
	HitCurrent(page Key_t, g []Gauge) (err error)
}

type GetPage_t[Key_t comparable] func(*http.Request) Key_t
type LogWrite_t func(ctx context.Context, format string, args ...interface{})
type LogRead_t func(ctx context.Context) []string

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
	for _, v := range self.log_read(ctx) {
		errors = v
		break
	}
	self.storage.HitEnd(counter, start, time.Now(), 1, strconv.FormatInt(int64(writer.status_code), 10), errors)
}
