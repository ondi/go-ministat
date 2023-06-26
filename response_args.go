//
//
//

package ministat

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
)

var TRIM = map[byte]bool{
	'\r': true,
	'\n': true,
	'\t': true,
	'\v': true,
}

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

func TrimRight(in []byte, tr map[byte]bool) []byte {
	pos := len(in)
	for pos > 0 {
		if tr[in[pos-1]] {
			pos--
		} else {
			break
		}
	}
	return in[:pos]
}

type Copy_t struct {
	Buf   bytes.Buffer
	Limit int
}

func (self *Copy_t) Write(p []byte) (n int, err error) {
	if n = self.Limit - self.Buf.Len(); n > len(p) {
		n, err = self.Buf.Write(p)
	} else {
		n, err = self.Buf.Write(p[:n])
	}
	return
}

type NoCopy_t struct{}

func (NoCopy_t) Write(p []byte) (n int, err error) {
	return
}
