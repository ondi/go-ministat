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

func KeyValues[Value_t any](m *Median_t[Value_t], ts time.Time, cmp Compare_t[Value_t]) (res []string) {
	m.Range(ts, cmp, func(k int, v Mapped_t[Value_t]) bool {
		res = append(res, fmt.Sprintf("(%v,%v)", k, v.Data))
		return true
	})
	return
}

func Keys[Value_t any](m *Median_t[Value_t], ts time.Time, cmp Compare_t[Value_t]) (res []int) {
	m.Range(ts, cmp, func(k int, v Mapped_t[Value_t]) bool {
		res = append(res, k)
		return true
	})
	return
}

func RealMedian[Value_t any](m *Median_t[Value_t], ts time.Time, cmp Compare_t[Value_t]) (key int, value Value_t) {
	_, size := m.Median(ts, cmp)
	half := size / 2
	m.Range(ts, cmp, func(k int, v Mapped_t[Value_t]) bool {
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

func check_sorted[Value_t any](m *Median_t[Value_t], ts time.Time, cmp Compare_t[Value_t]) (res string) {
	var prev_set bool
	var prev_value Value_t
	// do not evict
	m.Range(ts.Add(-time.Hour), cmp, func(k int, v Mapped_t[Value_t]) bool {
		if prev_set {
			if cmp(prev_value, v.Data) > 0 {
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

func debug_state[Value_t any](m *Median_t[Value_t], ts time.Time, cmp Compare_t[Value_t]) (res string) {
	if res = check_sorted(m, ts, cmp); len(res) > 0 {
		return
	}

	if m.cx.Size() > 0 && (m.left < 0 || m.right < 0 || m.left+m.right != m.cx.Size()-1 || m.left > m.right+1 || m.right > m.left+1) {
		res = fmt.Sprintf("SIZE CHECK: size=%v, left=%v, right=%v", m.cx.Size(), m.left, m.right)
		return
	}

	count := m.left
	// do not evict
	m.Range(ts.Add(-time.Hour), cmp, func(k int, v Mapped_t[Value_t]) bool {
		if cmp(v.Data, m.median.Value.Data) > 0 {
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
	m := NewMedian[int](10, 10*time.Second)
	for i := 0; i < 1000; i++ {
		m.Add(ts, 10, Cmp1)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts, Cmp1)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _ := m.Median(ts, Cmp1)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median20(t *testing.T) {
	m := NewMedian[int](11, 10*time.Second)
	for i := 0; i < 1000; i++ {
		m.Add(ts, i, Cmp1)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts, Cmp1)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _ := m.Median(ts, Cmp1)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median30(t *testing.T) {
	m := NewMedian[int](10, 10*time.Second)
	for i := 1000; i > 0; i-- {
		m.Add(ts, i, Cmp1)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %v %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts, Cmp1)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _ := m.Median(ts, Cmp1)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median40(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	m := NewMedian[int](21, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rand.Intn(1000), Cmp1)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %02d %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts, Cmp1)
	t.Logf("REAL MEDIAN: %v %v", k, v)
	median, _ := m.Median(ts, Cmp1)
	assert.Assert(t, median == v, fmt.Sprintf("TEST=%v, REAL=%v", median, v))
}

func Test_median50(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	size := 21
	m := NewMedian[int](size, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rand.Intn(1000), Cmp1)
		// m.Add(100, Cmp1)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %02d %v", key, value.Data)
		return true
	})

	k, v := RealMedian(m, ts, Cmp1)
	median, _ := m.Median(ts, Cmp1)
	t.Logf("REAL MEDIAN: %v %v, median=%v", k, v, median)

	for i := 0; i < size; i++ {
		begin := m.begin()
		t.Logf("REMOVE: %v", begin)
		it, ok := m.cx.Find(begin)
		assert.Assert(t, ok)
		m.remove(it, Cmp1)

		m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
			t.Logf("RANGE: %02d %v", key, value.Data)
			return true
		})

		t.Logf("MEDIAN: size=%v, left=%v, right=%v, mkey=%v, mvalue=%v", m.cx.Size(), m.left, m.right, m.median.Key, m.median.Value.Data)

		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
		begin++
		if begin >= m.limit {
			begin = 0
		}
	}
}

func Test_median60(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	size := 100
	m := NewMedian[int](size, 10*time.Second)
	for i := 0; i < 20000; i++ {
		m.Add(ts, rand.Intn(1000), Cmp1)
		ts = ts.Add(500 * time.Millisecond)
		check := debug_state(m, ts, Cmp1)
		assert.Assert(t, len(check) == 0, check)
	}

	m.Range(ts, Cmp1, func(key int, value Mapped_t[int]) bool {
		t.Logf("RANGE: %02d %v", key, value.Data)
		return true
	})
}
