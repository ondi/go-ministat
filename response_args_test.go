//
//
//

package ministat

import (
	"bytes"
	"testing"

	"gotest.tools/assert"
)

func Test_Args01(t *testing.T) {
	var buf bytes.Buffer
	cw := LimitWriter_t{Buf: &buf, Limit: 5}

	n, _ := cw.Write([]byte("123"))
	assert.Assert(t, n == 3, n)

	n, _ = cw.Write([]byte("456"))
	assert.Assert(t, n == 2, n)
}

func Test_Args02(t *testing.T) {
	var buf bytes.Buffer
	cw := LimitWriter_t{Buf: &buf, Limit: 0}

	n, _ := cw.Write([]byte("123"))
	assert.Assert(t, n == 0, n)

	n, _ = cw.Write([]byte("456"))
	assert.Assert(t, n == 0, n)
}
