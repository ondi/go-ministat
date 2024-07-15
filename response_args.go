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
		for ; self.Limit > 0; self.Limit-- {
			if r, _ := utf8.DecodeLastRune(p[:self.Limit]); r != utf8.RuneError {
				break
			}
		}
		n, err = self.Buf.Write(p[:self.Limit])
	}
	self.Limit -= n
	return
}
