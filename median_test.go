//
// go test -v -run Test_median10
//

package ministat

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gotest.tools/assert"
)

func Cmp1(a, b int) int {
	return a - b
}

func Test_median10(t *testing.T) {
	m := NewMedian[int](10)
	for i := 0; i < 100; i++ {
		m.Add(10, Cmp1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}

func Test_median20(t *testing.T) {
	m := NewMedian[int](10)
	for i := 0; i < 100; i++ {
		m.Add(i, Cmp1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}

func Test_median30(t *testing.T) {
	m := NewMedian[int](10)
	for i := 100; i > 0; i-- {
		m.Add(i, Cmp1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}

func Test_median40(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	m := NewMedian[int](21)
	for i := 0; i < 12345; i++ {
		m.Add(rand.Intn(1000), Cmp1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %02d %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}
