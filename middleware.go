//
//
//

package ministat

import (
	"net/http"
	"time"
)

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
	name := self.page_name(r)
	counter := self.storage.MetricBegin(name, start)

	writer := ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}

	r, ok := self.online.MinistatBegin(&writer, r, name, counter.Online)
	if ok {
		self.next.ServeHTTP(&writer, r)
	}

	diff := time.Since(start)
	self.online.MinistatEnd(r, name, writer.status_code, diff)

	self.storage.MetricEnd(counter, diff, 1, writer.status_code)
}
