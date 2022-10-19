//
// go test -run Test_median40 -v
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

// func RealMedian[Value_t any](m *Median_t[Value_t]) (key int64, value Value_t) {
func RealMedian(m *Median_t[int]) (key int64, value int) {
	half := m.Size() / 2
	m.Range(func(k int64, v int) bool {
		key = k
		value = v
		half--
		if half >= 0 {
			return true
		}
		return false
	})
	return
}

// func DebugLR[Value_t any](m *Median_t[Value_t]) string {
func DebugLR(m *Median_t[int]) (res string) {
	left, right, mkey, mvalue, size := m.DebugLR()
	count := left
	m.Range(func(k int64, v int) bool {
		count--
		if count >= 0 {
			return true
		}
		if k != mkey {
			res = fmt.Sprintf("MEDIAN CHECK: left=%v, right=%v, check=(%v,%v), median=(%v,%v), size=%v", left, right, k, v, mkey, mvalue, size)
		}
		return false
	})
	return
}

func Test_median10(t *testing.T) {
	m := NewMedian[int](10)
	for i := 0; i < 100; i++ {
		m.Add(10, Cmp1)
		check := DebugLR(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	_, v := RealMedian(m)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median20(t *testing.T) {
	m := NewMedian[int](10)
	for i := 0; i < 100; i++ {
		m.Add(i, Cmp1)
		check := DebugLR(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	_, v := RealMedian(m)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median30(t *testing.T) {
	m := NewMedian[int](10)
	for i := 100; i > 0; i-- {
		m.Add(i, Cmp1)
		check := DebugLR(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})
	_, v := RealMedian(m)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median40(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	m := NewMedian[int](21)
	for i := 0; i < 12345; i++ {
		m.Add(rand.Intn(1000), Cmp1)
		check := DebugLR(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int64, value int) bool {
		t.Logf("RANGE: %02d %v", key, value)
		return true
	})

	_, v := RealMedian(m)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}
