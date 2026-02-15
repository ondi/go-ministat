//
//
//

package ministat

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ondi/go-tst"
)

type cors_middleware_t struct {
	handler http.Handler
}

func NewCors(handler http.Handler) *cors_middleware_t {
	return &cors_middleware_t{
		handler: handler,
	}
}

func (self *cors_middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); len(origin) > 0 {
		w.Header().Add("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
	}
	if r.Method == http.MethodOptions {
		return
	}
	self.handler.ServeHTTP(w, r)
}

type timeout_middleware_t struct {
	routes  *tst.Tree3_t[time.Duration]
	handler http.Handler
	cause   error
}

func NewCtxTimeout(handler http.Handler, cause string, routes map[string]time.Duration) (self *timeout_middleware_t) {
	self = &timeout_middleware_t{
		handler: handler,
		cause:   errors.New(cause),
		routes:  tst.NewTree3[time.Duration](),
	}
	for k, v := range routes {
		self.routes.Add(k, v)
	}
	return
}

func (self *timeout_middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if value, _, found := self.routes.Search(r.URL.Path); found > 0 {
		ctx, cancel := context.WithTimeoutCause(r.Context(), value, self.cause)
		defer cancel()
		r = r.WithContext(ctx)
	}
	self.handler.ServeHTTP(w, r)
}
