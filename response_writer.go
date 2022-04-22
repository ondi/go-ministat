//
//
//

package ministat

import (
	"io"
	"net/http"
	"path"
	"runtime"

	"github.com/ondi/go-log"
	"github.com/ondi/go-tst"
)

type HttpWriter_t struct {
	http.ResponseWriter
	file        string
	line        int
	status_code int
}

func (self *HttpWriter_t) WriteHeader(status_code int) {
	_, self.file, self.line, _ = runtime.Caller(4)
	self.status_code = status_code
	self.ResponseWriter.WriteHeader(status_code)
}

type Writer_t struct {
	HttpWriter_t
	CopyWriter_t
}

func (self *Writer_t) Write(p []byte) (int, error) {
	self.CopyWriter_t.Write(p)
	return self.HttpWriter_t.Write(p)
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

type ResponseWriter_t struct {
	next    http.Handler
	exclude *tst.Tree1_t
}

func NewResponseWriter(next http.Handler, excluse []string) (self *ResponseWriter_t) {
	self = &ResponseWriter_t{
		next:    next,
		exclude: &tst.Tree1_t{},
	}
	for _, v := range excluse {
		self.exclude.Add(v, 1)
	}
	return
}

func (self *ResponseWriter_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reader := Reader_t{ReadCloser: r.Body}
	r.Body = &reader
	writer := Writer_t{HttpWriter_t: HttpWriter_t{ResponseWriter: w, status_code: http.StatusOK}}
	value := self.exclude.Search(r.URL.Path)
	if value == nil {
		log.TraceCtx(r.Context(), "REQUEST: %v", r.URL.String())
	}
	self.next.ServeHTTP(&writer, r)
	if value == nil {
		var errors []string
		if v := log.ContextGet(r.Context()); v != nil {
			errors = v.Values()
		}
		log.TraceCtx(r.Context(), "%s:%d RESPONSE: %s %d req='%s', resp='%s', errors=%s", path.Base(writer.file), writer.line, r.URL.String(), writer.status_code, reader.Data(), writer.Data(), errors)
	}
}
