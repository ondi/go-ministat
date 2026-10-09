//
// go test -run Test_average10 -v -count=1
//

package ministat

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func Test_average10(t *testing.T) {
	ts := time.Now()
	var res int
	var size int64
	m := NewAverage[int](100, 1000*time.Millisecond)
	for i := 0; i < 1000; i++ {
		ts = ts.Add(100 * time.Millisecond)
		res, size = m.Add(ts, i)
		t.Logf("res=%v, size=%v", res, size)
	}
	assert.Assert(t, res == 994, res)
	assert.Assert(t, size == 10, size)
}
