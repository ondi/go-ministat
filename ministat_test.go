//
//
//

package ministat

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ondi/go-unique"
	"gotest.tools/assert"
)

type Evict_t struct {
	t   *testing.T
	str string
}

func (self *Evict_t) Value(key interface{}, value unique.Counter) bool {
	self.t.Logf("EVICT: %v", key)
	assert.Assert(self.t, strings.Contains(key.(string), self.str), key)
	return true
}

func (self *Evict_t) Evict(f func(f func(key interface{}, value unique.Counter) bool)) {
	f(self.Value)
}

func Test_Evict01(t *testing.T) {
	s := NewStorage(0, 10, time.Second, (&Evict_t{t: t, str: "test1"}).Evict)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test1."+strconv.FormatInt(i, 10), ts)
	}
}

func Test_Evict02(t *testing.T) {
	s := NewStorage(1, 10, time.Second, (&Evict_t{t: t, str: "test2"}).Evict)

	ts := time.Now()
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test2."+strconv.FormatInt(i, 10), ts)
	}

	ts = ts.Add(time.Second)
	for i := int64(0); i < 10; i++ {
		s.MetricBegin("test3."+strconv.FormatInt(i, 10), ts)
	}
}