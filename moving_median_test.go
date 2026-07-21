//
// go test -run Test_median40 -v -count=1
//

package ministat

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gotest.tools/assert"
)

func KeyValues[Value_t Number](m *Median_t[Value_t], ts time.Time) (res []string) {
	m.range_test(ts, func(k int, v MedianMapped_t[Value_t]) bool {
		res = append(res, fmt.Sprintf("(%v,%v)", k, v.Data))
		return true
	})
	return
}

func Keys[Value_t Number](m *Median_t[Value_t], ts time.Time) (res []int) {
	m.range_test(ts, func(k int, v MedianMapped_t[Value_t]) bool {
		res = append(res, k)
		return true
	})
	return
}

func RealMedian[Value_t Number](m *Median_t[Value_t], ts time.Time) (key int, value Value_t) {
	_, _, _, size := m.Value(ts)
	half := size / 2
	m.range_test(ts, func(k int, v MedianMapped_t[Value_t]) bool {
		key = k
		value = v.Data
		half--
		if half >= 0 {
			return true
		}
		return false
	})
	return
}

func check_sorted[Value_t Number](m *Median_t[Value_t], ts time.Time) (res string) {
	var prev_set bool
	var prev_value Value_t
	// do not evict
	m.range_test(ts.Add(-time.Hour), func(k int, v MedianMapped_t[Value_t]) bool {
		if prev_set {
			if prev_value > v.Data {
				res = fmt.Sprintf("SORT CHECK: %v %v", prev_value, v)
				return false
			}
		}
		prev_value = v.Data
		prev_set = true
		return true
	})
	return
}

func debug_state[Value_t Number](m *Median_t[Value_t], ts time.Time) (res string) {
	if res = check_sorted(m, ts); len(res) > 0 {
		return
	}

	if m.cx.Size() > 0 && (m.left < 0 || m.right < 0 || m.left+m.right != m.cx.Size()-1 || m.left > m.right+1 || m.right > m.left+1) {
		res = fmt.Sprintf("SIZE CHECK: size=%v, left=%v, right=%v", m.cx.Size(), m.left, m.right)
		return
	}

	count := m.left
	// do not evict
	m.range_test(ts.Add(-time.Hour), func(k int, v MedianMapped_t[Value_t]) bool {
		if v.Data > m.median.Value.Data {
			res = fmt.Sprintf("MEDIAN VALUE: size=%v, left=%v, right=%v, check=(%v,%v), median=(%v,%v)", m.cx.Size(), m.left, m.right, k, v, m.median.Key, m.median.Value.Data)
			return false
		}
		count--
		if count >= 0 {
			return true
		}
		if k != m.median.Key {
			res = fmt.Sprintf("MEDIAN CHECK: size=%v, left=%v, right=%v, check=(%v,%v), median=(%v,%v)", m.cx.Size(), m.left, m.right, k, v, m.median.Key, m.median.Value.Data)
		}
		return false
	})
	return
}

func Test_median10(t *testing.T) {
	ts := time.Now()
	m := NewMedian[int](10, 10*time.Second)
	for i := 0; i < 1000; i++ {
		m.Add(ts, 10)
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _, _, _ := m.Value(ts)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median20(t *testing.T) {
	ts := time.Now()
	m := NewMedian[int](11, 10*time.Second)
	for i := 0; i < 1000; i++ {
		m.Add(ts, i)
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _, _, _ := m.Value(ts)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median30(t *testing.T) {
	ts := time.Now()
	m := NewMedian[int](10, 10*time.Second)
	for i := 1000; i > 0; i-- {
		m.Add(ts, i)
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _, _, _ := m.Value(ts)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median40(t *testing.T) {
	ts := time.Now()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := NewMedian[int](21, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rnd.Intn(1000))
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %02d %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _, _, _ := m.Value(ts)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median50(t *testing.T) {
	size := 21
	ts := time.Now()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := NewMedian[int](size, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rnd.Intn(1000))
		// m.Add(100, )
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %02d %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts)
	median, _, _, _ := m.Value(ts)
	t.Logf("REAL MEDIAN: %v %v, median=%v", k, v, median)

	for i := 0; i < size; i++ {
		begin := m.begin()
		t.Logf("REMOVE: %v", begin)
		it, ok := m.cx.Find(begin)
		assert.Assert(t, ok)
		m.remove(it)

		m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
			t.Logf("RANGE: %02d %v", key, value.Data)
			return true
		})

		t.Logf("MEDIAN: size=%v, left=%v, right=%v, mkey=%v, mvalue=%v", m.cx.Size(), m.left, m.right, m.median.Key, m.median.Value.Data)

		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
		begin++
		if begin >= m.limit {
			begin = 0
		}
	}
}

func Test_median60(t *testing.T) {
	size := 100
	ts := time.Now()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := NewMedian[int](size, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rnd.Intn(1000))
		ts = ts.Add(500 * time.Millisecond)
		check := debug_state(m, ts)
		assert.Assert(t, len(check) == 0, check)
	}

	m.range_test(ts, func(key int, value MedianMapped_t[int]) bool {
		t.Logf("RANGE: %02d %v %v", key, value.Data, value.Ts.Sub(ts))
		return true
	})
}
