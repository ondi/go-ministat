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

func (self *CopyWriter_t) TrimRight() []byte {
	for self.written > 0 {
		if TRIM[self.buf[self.written-1]] {
			self.written--
		} else {
			break
		}
	}
	return self.buf[:self.written]
}

func (self *CopyWriter_t) Bytes() []byte {
	return self.buf[:self.written]
}

func (self *CopyWriter_t) Args(args ...interface{}) {
	var n int
	for i, v := range args {
		if i > 0 {
			fmt.Fprintf(self, ",")
		}
		if temp, _ := v.(interface{ Value() (driver.Value, error) }); temp != nil {
			if value, err := temp.Value(); err == nil {
				v = value
			}
		}
		switch data := v.(type) {
		case []uint8:
			n, _ = fmt.Fprintf(self, "%s", data)
		default:
			n, _ = fmt.Fprintf(self, "%+v", data)
		}
		if n == 0 {
			return
		}
	}
}

func (self *CopyWriter_t) Reset() {
	self.written = 0
}
