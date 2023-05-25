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
	origin := r.Header.Get("Origin")
	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	if r.Method == "OPTIONS" {
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	self.handler.ServeHTTP(w, r)
}
