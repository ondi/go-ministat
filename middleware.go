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
	GetName() string
	GetStatus() string
	GetValueInt64() int64
	GetValueString() string
	String() string
}

type Views[Key_t comparable] interface {
	HitCurrent(page Key_t, g []Gauge) (err error)
}

type GetPage_t[Key_t comparable] func(*http.Request) Key_t
type LogWrite_t func(ctx context.Context, format string, args ...interface{})
type LogGetErrors_t func(ctx context.Context) string

type Middleware_t[Key_t comparable] struct {
	storage        *Storage_t[Key_t]
	next_passed    http.Handler
	next_failed    http.Handler
	page_name      GetPage_t[Key_t]
	log_get_errors LogGetErrors_t
	views          Views[Key_t]
	pending_limit  int64
}

func NewMiddleware[Key_t comparable](storage *Storage_t[Key_t], next_passed http.Handler, next_failed http.Handler, log_get_errors LogGetErrors_t, views Views[Key_t], page_name GetPage_t[Key_t], pending_limit int64) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:        storage,
		next_passed:    next_passed,
		next_failed:    next_failed,
		page_name:      page_name,
		pending_limit:  pending_limit,
		log_get_errors: log_get_errors,
		views:          views,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending, _ := self.storage.HitBegin(page, ts)
	defer func() {
		self.storage.HitEnd(counter, ts, time.Now(), 1, strconv.FormatInt(int64(writer.status_code), 10), self.log_get_errors(r.Context()))
	}()
	if sampling > 0 && pending <= self.pending_limit {
		self.next_passed.ServeHTTP(&writer, r)
	} else {
		self.next_failed.ServeHTTP(&writer, r)
	}
}
