//
//
//

package ministat

import (
	"strconv"
	"testing"
	"time"
)

func Test_Evict01(t *testing.T) {
	s := NewStorage(0, 10, time.Second, NoState_t{})

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test1-"+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second, NoState_t{})

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test2-"+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test3-"+strconv.FormatInt(i, 10), ts)
	}
}
