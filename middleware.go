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
	GetValueFloat64() float64
	String() string
}

type Views[Key_t comparable] interface {
	HitCurrent(page Key_t, g []Gauge) (err error)
}

type GetPage_t[Key_t comparable] func(*http.Request) Key_t
type GetErrors_t func(ctx context.Context, out map[string]int64)
type GetComments_t func(ctx context.Context, out map[string]string)
type LogWrite_t func(ctx context.Context, format string, args ...any)

type Middleware_t[Key_t comparable] struct {
	storage       *Storage_t[Key_t]
	next_passed   http.Handler
	next_failed   http.Handler
	page_name     GetPage_t[Key_t]
	views         Views[Key_t]
	pending_limit int64
	errors        []GetErrors_t
}

func NewMiddleware[Key_t comparable](storage *Storage_t[Key_t], next_passed http.Handler, next_failed http.Handler, views Views[Key_t], page_name GetPage_t[Key_t], pending_limit int64, errors ...GetErrors_t) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:       storage,
		next_passed:   next_passed,
		next_failed:   next_failed,
		page_name:     page_name,
		views:         views,
		pending_limit: pending_limit,
		errors:        errors,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending, _ := self.storage.HitBegin(page, ts)
	defer func() {
		errors := map[string]int64{}
		for _, v := range self.errors {
			v(r.Context(), errors)
		}
		self.storage.HitEnd(counter, ts, time.Now(), map[string]int64{strconv.FormatInt(int64(writer.status_code), 10): 1}, errors)
	}()
	if sampling > 0 && pending <= self.pending_limit {
		self.next_passed.ServeHTTP(&writer, r)
	} else {
		self.next_failed.ServeHTTP(&writer, r)
	}
}
