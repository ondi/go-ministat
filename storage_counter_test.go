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
	Views
	t     *testing.T
	check string
}

func (self *EvictTest_t) MinistatEvict(key string, DurationSum time.Duration, DurationNum time.Duration) (err error) {
	self.t.Logf("EVICT: %v", key)
	assert.Assert(self.t, strings.Contains(key, self.check), key)
	return
}

func Test_Evict01(t *testing.T) {
	s := NewStorage(0, 10, time.Second, &EvictTest_t{t: t, check: "test1"}, NoState_t{})

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test1-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second, &EvictTest_t{t: t, check: "test2"}, NoState_t{})

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test2-"+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test3-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Args01(t *testing.T) {
	var cw CopyWriter
	cw = NoCopyWriter_t{}
	cw.Truncate(0)
}

func Test_Args02(t *testing.T) {
	var cw CopyWriter
	cw = &CopyWriter_t{}
	cw.Truncate(0)
}
