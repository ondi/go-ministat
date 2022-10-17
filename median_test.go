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

func Compare1(a, b int) int {
	return a - b
}

func Test_median10(t *testing.T) {
	input := []int{100, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	m := NewMedian[int](10)
	for i := 0; i < 10; i++ {
		m.Add(input[i], Compare1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}

func Test_median20(t *testing.T) {
	input := []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	m := NewMedian[int](10)
	for i := 0; i < 10; i++ {
		m.Add(input[i], Compare1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}

func Test_median30(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	m := NewMedian[int](20)
	for i := 0; i < 10; i++ {
		m.Add(rand.Intn(1000), Compare1)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %02d %v", key, value)
		return true
	})

	assert.Assert(t, m.Median() == m.RealMedian().Value, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), m.RealMedian().Value))
}
