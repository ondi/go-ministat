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

func (self *ResponseWriter_t) Flush() {
	if h, ok := self.ResponseWriter.(http.Flusher); ok {
		h.Flush()
	}
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
	next       http.Handler
	log_write  LogWrite_t
	req_limit  int
	resp_limit int
	exclude    *tst.Tree3_t[int]
	tags       TagsAll_t
}

func NewResponseLogger(next http.Handler, log_write LogWrite_t, req_limit int, resp_limit int, excluse []string, tags TagsAll_t) (self *ResponseLogger_t) {
	self = &ResponseLogger_t{
		next:       next,
		log_write:  log_write,
		req_limit:  req_limit,
		resp_limit: resp_limit,
		exclude:    tst.NewTree3[int](),
		tags:       tags,
	}
	for _, v := range excluse {
		self.exclude.Add(v, 1)
	}
	return
}

func (self *ResponseLogger_t) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var writer_buf, reader_buf bytes.Buffer
	writer := Writer_t{ResponseWriter_t: ResponseWriter_t{ResponseWriter: w, status_code: http.StatusOK}, LimitWriter_t: LimitWriter_t{Buf: &writer_buf, Limit: self.resp_limit}}
	reader := Reader_t{ReadCloser: r.Body, LimitWriter_t: LimitWriter_t{Buf: &reader_buf, Limit: self.req_limit}}
	r.Body = &reader
	_, _, found := self.exclude.Search(r.URL.Path)
	if found == 0 {
		self.log_write(r.Context(), "REQUEST: %s", r.URL.String())
	}
	self.next.ServeHTTP(&writer, r)
	if found == 0 {
		tags := map[string]map[string]string{}
		if self.tags != nil {
			self.tags(r.Context(), tags)
		}
		self.log_write(r.Context(), "RESPONSE: %s, status=%d, tags=%+v, resp=%#q, req=%#q",
			r.URL.String(), writer.status_code, tags, writer_buf.Bytes(), reader_buf.Bytes())
	}
}
