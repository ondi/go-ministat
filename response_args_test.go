//
//
//

package ministat

import (
	"testing"

	"gotest.tools/assert"
)

func Test_Args01(t *testing.T) {
	cw := LimitWriter_t{Limit: 5}

	n, _ := cw.Write([]byte("123"))
	assert.Assert(t, n == 3, n)

	n, _ = cw.Write([]byte("456"))
	assert.Assert(t, n == 2, n)
}

func Test_Args02(t *testing.T) {
	cw := LimitWriter_t{Limit: 0}

	n, _ := cw.Write([]byte("123"))
	assert.Assert(t, n == 0, n)

	n, _ = cw.Write([]byte("456"))
	assert.Assert(t, n == 0, n)
}
