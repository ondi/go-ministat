//
//
//

package ministat

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func Test_average10(t *testing.T) {
	m := NewAverage[int](10, 10*time.Second)
	for i := 0; i < 1000; i++ {
		_, size := m.Add(ts, 10)
		assert.Assert(t, size > 0, size)
	}
}
