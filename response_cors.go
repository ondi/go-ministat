//
//
//

package ministat

import "net/http"

type Cors_t struct {
	Handler http.Handler
}

func (self *Cors_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	self.Handler.ServeHTTP(w, r)
}
