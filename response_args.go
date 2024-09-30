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

type ValueFunc interface {
	Value() (driver.Value, error)
}

func Args(out io.Writer, args ...interface{}) {
	for i, v := range args {
		if i > 0 {
			fmt.Fprintf(out, ",")
		}
		if temp, ok := v.(ValueFunc); ok {
			if value, err := temp.Value(); err == nil {
				v = value
			}
		} else if temp, ok := v.(*ValueFunc); ok {
			if value, err := (*temp).Value(); err == nil {
				v = value
			}
		}
		switch data := v.(type) {
		case []uint8, json.RawMessage:
			fmt.Fprintf(out, "'%s'", data)
		case string:
			fmt.Fprintf(out, "'%s'", data)
		default:
			fmt.Fprintf(out, "%+v", data)
		}
	}
}

type LimitWriter_t struct {
	Buf   io.Writer
	Limit int
}

func (self *LimitWriter_t) Write(p []byte) (n int, err error) {
	if self.Limit >= len(p) {
		n, err = self.Buf.Write(p)
	} else {
		n, err = self.Buf.Write(TrimBadRune(p, self.Limit))
	}
	self.Limit -= n
	return
}

func TrimBadRune(in []byte, len_in int) []byte {
	for ; len_in > 0; len_in-- {
		if r, _ := utf8.DecodeLastRune(in[:len_in]); r != utf8.RuneError {
			break
		}
	}
	return in[:len_in]
}
