//
//
//

package ministat

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus_t struct {
	pageRequest           *prometheus.CounterVec
	pagePending           *prometheus.GaugeVec
	pageProcessed         *prometheus.CounterVec
	pageError             *prometheus.CounterVec
	pageLatencyMedian     *prometheus.GaugeVec
	pageLatencyMedianSize *prometheus.GaugeVec
}

// import "github.com/prometheus/client_golang/prometheus/promhttp"
// mux.Handle("/debug/metrics", promhttp.Handler())
func NewPrometheusViews(prefix string) (self *Prometheus_t, err error) {
	self = &Prometheus_t{
		pageRequest:           prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "request_count"}, []string{"page"}),
		pagePending:           prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "pending_sum"}, []string{"page"}),
		pageProcessed:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "payload_processed"}, []string{"page", "status"}),
		pageError:             prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "payload_error"}, []string{"page", "error"}),
		pageLatencyMedian:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency_median"}, []string{"page"}),
		pageLatencyMedianSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency_median_size"}, []string{"page"}),
	}
	prometheus.Register(self.pageRequest)
	prometheus.Register(self.pagePending)
	prometheus.Register(self.pageProcessed)
	prometheus.Register(self.pageError)
	prometheus.Register(self.pageLatencyMedian)
	prometheus.Register(self.pageLatencyMedianSize)
	return
}

func (self *Prometheus_t) HitBegin(ctx context.Context, page string) (err error) {
	_request, err := self.pageRequest.GetMetricWith(prometheus.Labels{"page": page})
	if err != nil {
		return
	}
	_pending, err := self.pagePending.GetMetricWith(prometheus.Labels{"page": page})
	if err != nil {
		return
	}
	_request.Add(1)
	_pending.Add(1)
	return
}

func (self *Prometheus_t) HitEnd(ctx context.Context, page string, median time.Duration, median_size int, processed int64, status int, errors string) (err error) {
	_pending, err := self.pagePending.GetMetricWith(prometheus.Labels{"page": page})
	if err != nil {
		return
	}
	_processed, err := self.pageProcessed.GetMetricWith(prometheus.Labels{"page": page, "status": strconv.FormatInt(int64(status), 10)})
	if err != nil {
		return
	}
	_latency, err := self.pageLatencyMedian.GetMetricWith(prometheus.Labels{"page": page})
	if err != nil {
		return
	}
	_latency_size, err := self.pageLatencyMedianSize.GetMetricWith(prometheus.Labels{"page": page})
	if err != nil {
		return
	}
	if len(errors) > 0 {
		var _error prometheus.Counter
		_error, err = self.pageError.GetMetricWith(prometheus.Labels{"page": page, "error": errors})
		if err != nil {
			return
		}
		_error.Add(float64(processed))
	}
	_pending.Add(-1)
	_processed.Add(float64(processed))
	_latency.Set(float64(median))
	_latency_size.Set(float64(median_size))
	return
}
