//
//
//

package ministat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

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

type Comment_t struct {
	Key   string
	Value string
}

type CommentList_t []Comment_t

func (self CommentList_t) String() string {
	var buf strings.Builder
	for i, v := range self {
		if i > 0 {
			buf.WriteString(",")
		}
		fmt.Fprintf(&buf, "%v=%v", v.Key, v.Value)
	}
	return buf.String()
}

type ResponseLogger_t struct {
	next        http.Handler
	log_write   LogWrite_t
	req_limit   int
	resp_limit  int
	exclude     *tst.Tree3_t[int]
	get_comment []GetComment_t
}

func NewResponseLogger(next http.Handler, log_write LogWrite_t, req_limit int, resp_limit int, excluse []string, get_comment ...GetComment_t) (self *ResponseLogger_t) {
	self = &ResponseLogger_t{
		next:        next,
		log_write:   log_write,
		req_limit:   req_limit,
		resp_limit:  resp_limit,
		exclude:     tst.NewTree3[int](),
		get_comment: get_comment,
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
		var comments CommentList_t
		for _, v := range self.get_comment {
			v(r.Context(), func(level_id int64, format string, args ...any) {
				for _, v2 := range args {
					if temp, ok := v2.(Comment_t); ok {
						comments = append(comments, temp)
					}
				}
			})
		}
		self.log_write(r.Context(), "RESPONSE: %s, status=%d, comments=%+v, resp=%#q, req=%#q",
			r.URL.String(), writer.status_code, comments, writer_buf.Bytes(), reader_buf.Bytes())
	}
}
