//
//
//

package ministat

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/ondi/go-log"
	"github.com/ondi/go-tst"
)

type ResponseWriter_t struct {
	http.ResponseWriter
	status_code int
}

func (self *ResponseWriter_t) WriteHeader(status_code int) {
	self.ResponseWriter.WriteHeader(status_code)
	self.status_code = status_code
}

func (self *ResponseWriter_t) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := self.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("not a http.Hijacker")
}

type Writer_t struct {
	ResponseWriter_t
	CopyWriter_t
}

func (self *Writer_t) Write(p []byte) (n int, err error) {
	n, err = self.ResponseWriter_t.Write(p)
	self.CopyWriter_t.Write(p[:n])
	return
}

type Reader_t struct {
	io.ReadCloser
	CopyWriter_t
}

func (self *Reader_t) Read(p []byte) (n int, err error) {
	n, err = self.ReadCloser.Read(p)
	self.CopyWriter_t.Write(p[:n])
	return
}

type ResponseLogger_t struct {
	next    http.Handler
	exclude *tst.Tree1_t[int]
}

func NewResponseLogger(next http.Handler, excluse []string) (self *ResponseLogger_t) {
	self = &ResponseLogger_t{
		next:    next,
		exclude: &tst.Tree1_t[int]{},
	}
	for _, v := range excluse {
		self.exclude.Add(v, 1)
	}
	return
}

func (self *ResponseLogger_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reader := Reader_t{ReadCloser: r.Body}
	r.Body = &reader
	writer := Writer_t{ResponseWriter_t: ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}}
	_, ok := self.exclude.Search(r.URL.Path)
	if !ok {
		log.TraceCtx(r.Context(), "REQUEST: %v", r.URL.String())
	}
	self.next.ServeHTTP(&writer, r)
	if !ok {
		var errors []string
		if v := log.ContextGet(r.Context()); v != nil {
			errors = v.Values()
		}
		log.TraceCtx(r.Context(), "%v RESPONSE: %d req='%s', resp='%s', errors=%s", r.URL.String(), writer.status_code, reader.TrimRight(), writer.TrimRight(), errors)
	}
}
