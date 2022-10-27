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

var ts = time.Now()

func Cmp1(a, b int) int {
	return a - b
}

func KeyValues[Value_t any](m *Median_t[Value_t]) (res []string) {
	m.Range(func(k int, v Value_t) bool {
		res = append(res, fmt.Sprintf("(%v,%v)", k, v))
		return true
	})
	return
}

func Keys[Value_t any](m *Median_t[Value_t]) (res []int) {
	m.Range(func(k int, v Value_t) bool {
		res = append(res, k)
		return true
	})
	return
}

func RealMedian[Value_t any](m *Median_t[Value_t]) (key int, value Value_t) {
	half := m.Size() / 2
	m.Range(func(k int, v Value_t) bool {
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

func debug_state[Value_t any](m *Median_t[Value_t]) (res string) {
	left, right, mkey, mvalue, size := m.debug_state()
	if size > 0 && (left < 0 || right < 0 || left+right != size-1) {
		res = fmt.Sprintf("SIZE: left=%v, right=%v, size=%v", left, right, size)
		return
	}
	count := left
	m.Range(func(k int, v Value_t) bool {
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
	for i := 0; i < 1000; i++ {
		m.Add(ts, 10, Cmp1)
		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	k, v := RealMedian(m)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median20(t *testing.T) {
	m := NewMedian[int](10)
	for i := 0; i < 1000; i++ {
		m.Add(ts, i, Cmp1)
		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	k, v := RealMedian(m)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median30(t *testing.T) {
	m := NewMedian[int](10)
	for i := 1000; i > 0; i-- {
		m.Add(ts, i, Cmp1)
		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int, value int) bool {
		t.Logf("RANGE: %v %v", key, value)
		return true
	})

	k, v := RealMedian(m)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median40(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	m := NewMedian[int](21)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rand.Intn(1000), Cmp1)
		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int, value int) bool {
		t.Logf("RANGE: %02d %v", key, value)
		return true
	})

	k, v := RealMedian(m)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	assert.Assert(t, m.Median() == v, fmt.Sprintf("TEST=%v, REAL=%v", m.Median(), v))
}

func Test_median50(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	size := 21
	m := NewMedian[int](size)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rand.Intn(1000), Cmp1)
		// m.Add(100, Cmp1)
		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(func(key int, value int) bool {
		t.Logf("RANGE: %02d %v", key, value)
		return true
	})

	k, v := RealMedian(m)
	t.Logf("REAL MEDIAN: %v %v, median=%v", k, v, m.Median())

	for i := 0; i < size; i++ {
		t.Logf("REMOVE: %v", i)
		m.debug_remove(i, Cmp1)

		m.Range(func(key int, value int) bool {
			t.Logf("RANGE: %02d %v", key, value)
			return true
		})

		left, right, mkey, mvalue, size := m.debug_state()
		t.Logf("MEDIAN: size=%v, left=%v, right=%v, mkey=%v, mvalue=%v", size, left, right, mkey, mvalue)

		check := debug_state(m)
		assert.Assert(t, len(check) == 0, check)
	}
}
