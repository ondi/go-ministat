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
	Copy_t
}

func (self *Writer_t) Write(p []byte) (n int, err error) {
	n, err = self.ResponseWriter_t.Write(p)
	self.Copy_t.Write(p[:n])
	return
}

type Reader_t struct {
	io.ReadCloser
	Copy_t
}

func (self *Reader_t) Read(p []byte) (n int, err error) {
	n, err = self.ReadCloser.Read(p)
	self.Copy_t.Write(p[:n])
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
	writer := Writer_t{ResponseWriter_t: ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}, Copy_t: Copy_t{Limit: 1024}}
	reader := Reader_t{ReadCloser: r.Body, Copy_t: Copy_t{Limit: 1024}}
	r.Body = &reader
	_, ok := self.exclude.Search(r.URL.Path)
	if !ok {
		self.log(r.Context(), "REQUEST: %v", r.URL.String())
	}
	self.next.ServeHTTP(&writer, r)
	if !ok {
		var sb bytes.Buffer
		self.errors(r.Context(), &sb)
		self.log(r.Context(), "%v RESPONSE: %d resp='%s', req='%s', errors=%s",
			r.URL.String(), writer.status_code, TrimRight(writer.Buf.Bytes(), TRIM), TrimRight(reader.Buf.Bytes(), TRIM), sb.String())
	}
}
