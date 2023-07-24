//
//
//

package ministat

import (
	"strconv"
	"testing"
	"time"

	"gotest.tools/assert"
)

func Test_Evict01(t *testing.T) {
	s := NewStorage(0, 10, time.Second)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test1-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test2-"+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test3-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Get01(t *testing.T) {
	s := NewStorage(1, 10, time.Second)

	ts := time.Now()
	s.MetricBegin("test1", ts)
	c, ok := s.MetricGet("test1", ts)
	assert.Assert(t, ok, ok)
	assert.Assert(t, c.Hits == 1, c.Hits)
}
