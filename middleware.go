//
//
//

package ministat

import (
	"net/http"
	"time"
)

type Online interface {
	MinistatContext(w http.ResponseWriter, r *http.Request, page string, online int64) (*http.Request, bool)
	MinistatBegin(r *http.Request, page string)
	MinistatEnd(r *http.Request, page string)
	MinistatDuration(r *http.Request, page string, status int, diff time.Duration)
}

type Middleware_t struct {
	storage   *Storage_t
	next      http.Handler
	page_name func(*http.Request) string
	online    Online
}

func NewMiddleware(storage *Storage_t, next http.Handler, page_name func(*http.Request) string, online Online) (self *Middleware_t) {
	self = &Middleware_t{
		storage:   storage,
		next:      next,
		page_name: page_name,
		online:    online,
	}
	return
}

func (self *Middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	page := self.page_name(r)
	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}

	counter, online, ref := self.storage.MetricBegin(page, start)

	r, ok := self.online.MinistatContext(&writer, r, page, online)

	if ref > 0 {
		self.online.MinistatBegin(r, page)
	}
	if ok {
		self.next.ServeHTTP(&writer, r)
	}
	if ref > 0 {
		self.online.MinistatEnd(r, page)
	}

	diff := time.Since(start)
	ref = self.storage.MetricEnd(counter, diff, 1, writer.status_code)

	if ref > 0 {
		self.online.MinistatDuration(r, page, writer.status_code, diff)
	}
}
