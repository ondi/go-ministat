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
	GetLevel() string
	GetTag() string
	GetValueInt64() int64
	GetValueFloat64() float64
	String() string
}

type Views[Key_t comparable] interface {
	HitCurrent(page Key_t, g []Gauge) (err error)
}

type GetPage_t[Key_t comparable] func(*http.Request) Key_t
type TagsCount_t func(ctx context.Context, out map[string]map[string]int64)
type TagsAll_t func(ctx context.Context, out map[string]map[string]string)
type LogWrite_t func(ctx context.Context, format string, args ...any)

type Middleware_t[Key_t comparable] struct {
	storage       *Storage_t[Key_t]
	next_passed   http.Handler
	next_failed   http.Handler
	page_name     GetPage_t[Key_t]
	views         Views[Key_t]
	pending_limit int64
	tags          TagsCount_t
	replace       map[int]Key_t
}

func NewMiddleware[Key_t comparable](
	storage *Storage_t[Key_t],
	next_passed http.Handler,
	next_failed http.Handler,
	views Views[Key_t],
	page_name GetPage_t[Key_t],
	pending_limit int64,
	tags TagsCount_t,
	replace map[int]Key_t,
) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:       storage,
		next_passed:   next_passed,
		next_failed:   next_failed,
		page_name:     page_name,
		views:         views,
		pending_limit: pending_limit,
		tags:          tags,
		replace:       replace,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending, _ := self.storage.HitBegin(page, ts)
	defer func() {
		tags := map[string]map[string]int64{}
		if self.tags != nil {
			self.tags(r.Context(), tags)
		}
		if tags["CODE"] == nil {
			tags["CODE"] = map[string]int64{}
		}
		tags["CODE"][strconv.FormatInt(int64(writer.status_code), 10)] = 1
		if temp, ok := self.replace[writer.status_code]; ok {
			self.storage.HitEnd(counter, ts, time.Now(), tags, []Key_t{page, temp})
		} else {
			self.storage.HitEnd(counter, ts, time.Now(), tags, nil)
		}
	}()
	if sampling > 0 && pending <= self.pending_limit {
		self.next_passed.ServeHTTP(&writer, r)
	} else {
		self.next_failed.ServeHTTP(&writer, r)
	}
}
