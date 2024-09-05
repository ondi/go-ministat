//
//
//

package ministat

import (
	"fmt"
	"time"
)

type GaugeInt64_t struct {
	Type   string `json:"type"`
	Result string `json:"result"`
	Value  int64  `json:"value"`
}

func (self GaugeInt64_t) GetType() string {
	return self.Type
}

func (self GaugeInt64_t) GetResult() string {
	return self.Result
}

func (self GaugeInt64_t) GetValueInt64() int64 {
	return self.Value
}

func (self GaugeInt64_t) String() string {
	if len(self.Result) > 0 {
		return fmt.Sprintf("{%s:%v %q}", self.Type, self.Value, self.Result)
	}
	return fmt.Sprintf("{%s:%v}", self.Type, self.Value)
}

type GaugeDuration_t struct {
	Type   string        `json:"type"`
	Result string        `json:"result"`
	Value  time.Duration `json:"value"`
}

func (self GaugeDuration_t) GetType() string {
	return self.Type
}

func (self GaugeDuration_t) GetResult() string {
	return self.Result
}

func (self GaugeDuration_t) GetValueInt64() int64 {
	return self.Value.Nanoseconds()
}

func (self GaugeDuration_t) String() string {
	if len(self.Result) > 0 {
		return fmt.Sprintf("{%s:%v %q}", self.Type, self.Value, self.Result)
	}
	return fmt.Sprintf("{%s:%v}", self.Type, self.Value)
}
