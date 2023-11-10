//
//
//

package ministat

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"

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
	LimitWriter_t
}

func (self *Writer_t) Write(p []byte) (n int, err error) {
	n, err = self.ResponseWriter_t.Write(p)
	self.LimitWriter_t.Write(p[:n])
	return
}

type Reader_t struct {
	io.ReadCloser
	LimitWriter_t
}

func (self *Reader_t) Read(p []byte) (n int, err error) {
	n, err = self.ReadCloser.Read(p)
	self.LimitWriter_t.Write(p[:n])
	return
}

type ResponseLogger_t struct {
	next    http.Handler
	log     LogCtx_t
	errors  GetErr_t
	exclude *tst.Tree1_t[int]
}

func NewResponseLogger(next http.Handler, log LogCtx_t, errors GetErr_t, excluse []string) (self *ResponseLogger_t) {
	self = &ResponseLogger_t{
		next:    next,
		log:     log,
		errors:  errors,
		exclude: &tst.Tree1_t[int]{},
	}
	for _, v := range excluse {
		self.exclude.Add(v, 1)
	}
	return
}

func (self *ResponseLogger_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var writer_buf, reader_buf bytes.Buffer
	writer := Writer_t{ResponseWriter_t: ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}, LimitWriter_t: LimitWriter_t{Out: &writer_buf, Limit: 1024}}
	reader := Reader_t{ReadCloser: r.Body, LimitWriter_t: LimitWriter_t{Out: &reader_buf, Limit: 1024}}
	r.Body = &reader
	_, ok := self.exclude.Search(r.URL.Path)
	if !ok {
		self.log(r.Context(), "REQUEST: %v", r.URL.String())
	}
	self.next.ServeHTTP(&writer, r)
	if !ok {
		var sb bytes.Buffer
		self.errors(r.Context(), &sb)
		self.log(r.Context(), "RESPONSE: %v status=%d resp='%s', req='%s', errors=%s",
			r.URL.String(), writer.status_code, bytes.TrimRight(writer_buf.Bytes(), "\r\n\t\v"), bytes.TrimRight(reader_buf.Bytes(), "\r\n\t\v"), sb.String())
	}
}
