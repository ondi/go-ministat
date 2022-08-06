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

var TooMany http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

func TrimValue(s string, out *strings.Builder) *strings.Builder {
	if len(s) > 255 {
		s = s[:255]
	}
	for _, r := range s {
		if r >= 0x20 && r <= 0x7e {
			out.WriteRune(r)
		}
	}
	return out
}

func NoErrors(ctx context.Context, sb *strings.Builder) *strings.Builder {
	return sb
}

type Middleware_t struct {
	storage   *Storage_t
	ok        http.Handler
	err       http.Handler
	errors    ErrGet_t
	log       ErrLog_t
	views     Views
	page_name func(*http.Request) string
}

type MiddlewareOptions func(self *Middleware_t)

func MiddlewarePageName(f func(*http.Request) string) MiddlewareOptions {
	return func(self *Middleware_t) {
		self.page_name = f
	}
}

func NewMiddleware(storage *Storage_t, ok http.Handler, err http.Handler, errors ErrGet_t, log ErrLog_t, views Views, opts ...MiddlewareOptions) (self *Middleware_t) {
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
	if c.Sampling == 0 || c.State == 1 {
		self.err.ServeHTTP(&writer, r)
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
		var sb1, sb2 strings.Builder
		self.errors(r.Context(), &sb1)
		if err = self.views.MinistatDuration(r.Context(), page, diff, 1, writer.status_code, TrimValue(sb1.String(), &sb2).String()); err != nil {
			self.log(r.Context(), "MINISTAT: %v %q", err, page)
		}
	}
}
