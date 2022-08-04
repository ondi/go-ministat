//
//
//

package ministat

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/ondi/go-log"
)

type LogCtx_t struct {
	Handler http.Handler
}

func (self *LogCtx_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String(), "ERROR")))
	self.Handler.ServeHTTP(w, r)
}
