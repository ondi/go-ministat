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
	s := NewStorage(0, 10, time.Second, NoEvict[string])

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.HitBegin("test1-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second, NoEvict[string])

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.HitBegin("test2-"+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.HitBegin("test3-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Get01(t *testing.T) {
	s := NewStorage(1, 10, time.Second, NoEvict[string])

	ts := time.Now()
	s.HitBegin("test1", ts)
	res, ok := s.HitGet(ts, "test1")
	assert.Assert(t, ok, ok)
	assert.Assert(t, res.GaugeLast[0].GetValueInt64() == 1, res.GaugeLast)
}
