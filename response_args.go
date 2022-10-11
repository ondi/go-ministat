//
//
//

package ministat

import (
	"database/sql/driver"
	"fmt"
)

var TRIM = map[byte]bool{
	'\r': true,
	'\n': true,
	'\t': true,
	'\v': true,
}

type CopyWriter interface {
	Write(p []byte) (n int, err error)
	Bytes() []byte
	Truncate(n int)
}

func Args(cw CopyWriter, args ...interface{}) {
	var n int
	for i, v := range args {
		if i > 0 {
			fmt.Fprintf(cw, ",")
		}
		if temp, _ := v.(interface{ Value() (driver.Value, error) }); temp != nil {
			if value, err := temp.Value(); err == nil {
				v = value
			}
		}
		switch data := v.(type) {
		case []uint8:
			n, _ = fmt.Fprintf(cw, "%s", data)
		default:
			n, _ = fmt.Fprintf(cw, "%+v", data)
		}
		if n == 0 {
			return
		}
	}
}

func TrimRight(cw CopyWriter) []byte {
	written := len(cw.Bytes())
	for written > 0 {
		if TRIM[cw.Bytes()[written-1]] {
			written--
		} else {
			break
		}
	}
	return cw.Bytes()[:written]
}

type CopyWriter_t struct {
	buf     [1024]byte
	written int
}

func (self *CopyWriter_t) Write(p []byte) (n int, err error) {
	if n = len(self.buf) - self.written; n > len(p) {
		_ = append(self.buf[:self.written], p...)
		self.written += len(p)
	} else {
		_ = append(self.buf[:self.written], p[:n]...)
		self.written += n
	}
	return
}

func (self *CopyWriter_t) Bytes() []byte {
	return self.buf[:self.written]
}

func (self *CopyWriter_t) Truncate(n int) {
	self.written = n
}

type NoCopyWriter_t struct{}

func (NoCopyWriter_t) Write(p []byte) (n int, err error) {
	return
}

func (NoCopyWriter_t) Bytes() (res []byte) {
	return
}

func (NoCopyWriter_t) Truncate(n int) {
	return
}
