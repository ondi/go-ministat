//
//
//

package ministat

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type SetCtx func(ctx context.Context, name string, levels string) context.Context

type errors_middleware_t struct {
	Handler http.Handler
	SetCtx  SetCtx
	Levels  string
}

func NewErrorsMiddleware(next http.Handler, set SetCtx, levels string) http.Handler {
	self := &errors_middleware_t{
		Handler: next,
		SetCtx:  set,
		Levels:  levels,
	}
	return self
}

func (self *errors_middleware_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(self.SetCtx(r.Context(), uuid.New().String(), self.Levels))
	self.Handler.ServeHTTP(w, r)
}
