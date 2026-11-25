//
//
//

package ministat

import (
	"context"
	"encoding/json"
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
type TagsCount_t func(ctx context.Context, f func(name string, level string, key string, count int64) bool)
type TagsAll_t func(ctx context.Context, f func(name string, level string, key string, value string) bool)

type Middleware_t[Key_t comparable] struct {
	storage       *Storage_t[Key_t]
	next_passed   http.Handler
	next_failed   http.Handler
	get_page      GetPage_t[Key_t]
	get_tags      TagsCount_t
	pending_limit int64
}

func NewMiddleware[Key_t comparable](
	storage *Storage_t[Key_t],
	next_passed http.Handler,
	next_failed http.Handler,
	get_page GetPage_t[Key_t],
	get_tags TagsCount_t,
	pending_limit int64,
) *Middleware_t[Key_t] {
	return &Middleware_t[Key_t]{
		storage:       storage,
		next_passed:   next_passed,
		next_failed:   next_failed,
		get_page:      get_page,
		get_tags:      get_tags,
		pending_limit: pending_limit,
	}
}

func (self *Middleware_t[Key_t]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	page := self.get_page(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}
	counter, sampling, pending, _ := self.storage.HitBegin(page, ts)
	defer func() {
		var tags []TagCount_t
		if self.get_tags != nil {
			self.get_tags(r.Context(), func(name string, level string, key string, value int64) bool {
				tags = append(tags, TagCount_t{Name: name, Level: level, Key: key, Count: value})
				return true
			})
		}
		tags = append(tags, TagCount_t{Name: "status", Level: "", Key: strconv.FormatInt(int64(writer.status_code), 10), Count: 1})
		self.storage.HitEnd(counter, ts, time.Now(), tags)
	}()
	if sampling > 0 && pending <= self.pending_limit {
		self.next_passed.ServeHTTP(&writer, r)
	} else {
		self.next_failed.ServeHTTP(&writer, r)
	}
}

type StatusBodyError_t struct {
	Code      int    `json:"code,omitzero"`
	ErrorCode int    `json:"errorCode,omitzero"`
	Debug     string `json:"debug,omitzero"`
	Message   string `json:"message,omitzero"`
}

type StatusBody_t struct {
	Error StatusBodyError_t `json:"error"`
}

func (self StatusBody_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(self.Error.Code)
	e := json.NewEncoder(w)
	e.SetIndent("", "   ")
	e.Encode(self)
}
