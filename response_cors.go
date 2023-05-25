//
//
//

package ministat

import "net/http"

type cors_t struct {
	handler http.Handler
	host    string
}

func NewCors(handler http.Handler, host string) *cors_t {
	return &cors_t{
		handler: handler,
		host:    host,
	}
}

func (self *cors_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Access-Control-Allow-Origin", self.host)
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Add("Access-Control-Allow-Origin", self.host)
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	self.handler.ServeHTTP(w, r)
}
