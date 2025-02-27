//
//
//

package ministat

import (
	"context"
	"errors"
	"net/http"
	"time"
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
	handler http.Handler
	timeout time.Duration
	cause   error
}

func NewCtxTimeout(handler http.Handler, timeout time.Duration, cause string) *timeout_middleware_t {
	return &timeout_middleware_t{
		handler: handler,
		timeout: timeout,
		cause:   errors.New(cause),
	}
}

func (self *timeout_middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeoutCause(r.Context(), self.timeout, self.cause)
	defer cancel()
	self.handler.ServeHTTP(w, r.WithContext(ctx))
}
