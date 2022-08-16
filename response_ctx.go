//
//
//

package ministat

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type SetCtx_t func(ctx context.Context, name string, levels string) context.Context

type log_ctx_t struct {
	Handler http.Handler
	SetCtx  SetCtx_t
}

func NewLogCtx(next http.Handler, set SetCtx_t) http.Handler {
	self := &log_ctx_t{
		Handler: next,
		SetCtx:  set,
	}
	return self
}

func (self *log_ctx_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(self.SetCtx(r.Context(), uuid.New().String(), "ERROR"))
	self.Handler.ServeHTTP(w, r)
}
