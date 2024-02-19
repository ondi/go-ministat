//
//
//

package ministat

import "net/http"

type cors_t struct {
	handler http.Handler
}

func NewCors(handler http.Handler) *cors_t {
	return &cors_t{
		handler: handler,
	}
}

func (self *cors_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		origin := r.Header.Get("Origin")
		if len(origin) == 0 {
			origin = "*"
		}
		w.Header().Add("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		return
	}
	self.handler.ServeHTTP(w, r)
}
