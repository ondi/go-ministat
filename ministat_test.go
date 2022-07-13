//
//
//

package ministat

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

type EvictTest_t struct {
	t   *testing.T
	str string
}

func (self *EvictTest_t) Evict(f func(f func(key string, value *Counter_t) bool)) {
	f(
		func(key string, value *Counter_t) bool {
			self.t.Logf("EVICT: %v", key)
			assert.Assert(self.t, strings.Contains(key, self.str), key)
			return true
		},
	)
}

func Test_Evict01(t *testing.T) {
	s := NewStorage(0, 10, time.Second, (&EvictTest_t{t: t, str: "test1"}).Evict)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test1."+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second, (&EvictTest_t{t: t, str: "test2"}).Evict)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test2."+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test3."+strconv.FormatInt(i, 10), ts)
	}
}
