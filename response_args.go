//
//
//

package ministat

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

func Args(out io.Writer, args ...interface{}) {
	var n int
	for i, v := range args {
		if i > 0 {
			fmt.Fprintf(out, ",")
		}
		if temp, _ := v.(interface{ Value() (driver.Value, error) }); temp != nil {
			if value, err := temp.Value(); err == nil {
				v = value
			}
		}
		switch data := v.(type) {
		case []uint8, json.RawMessage:
			n, _ = fmt.Fprintf(out, "%s", data)
		default:
			n, _ = fmt.Fprintf(out, "%+v", data)
		}
		if n == 0 {
			return
		}
	}
}

type LimitWriter_t struct {
	Out     io.Writer
	Limit   int
	written int
}

func (self *LimitWriter_t) Write(p []byte) (n int, err error) {
	if n = self.Limit - self.written; n > len(p) {
		n, err = self.Out.Write(p)
	} else {
		for ; n > 0; n-- {
			if r, _ := utf8.DecodeLastRune(p[:n]); r != utf8.RuneError {
				break
			}
		}
		n, err = self.Out.Write(p[:n])
	}
	self.written += n
	return
}
