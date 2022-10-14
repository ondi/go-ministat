//
//
//

package ministat

import (
	"testing"

	"github.com/ondi/go-cache"
	"gotest.tools/assert"
)

func MLess1(a, b *cache.Value_t[int64, int]) bool {
	return a.Value < b.Value
}

func Test_median10(t *testing.T) {
	m := NewMedian[int](10)
	m.Add(50, MLess1)
	m.Add(10, MLess1)
	m.Add(60, MLess1)
	m.Add(40, MLess1)
	m.Add(70, MLess1)
	m.Add(20, MLess1)

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	// assert.Assert(t, m.Size() == 4, m.Size())
	assert.Assert(t, m.Median() == 50, m.Median())
}
