//
// for Page_t only
//

package ministat

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus_t struct {
	Request           *prometheus.CounterVec
	Pending           *prometheus.GaugeVec
	Processed         *prometheus.CounterVec
	Error             *prometheus.CounterVec
	LatencyMedian     *prometheus.GaugeVec
	LatencyMedianSize *prometheus.GaugeVec
}

// import "github.com/prometheus/client_golang/prometheus/promhttp"
// mux.Handle("/debug/metrics", promhttp.Handler())
func NewPrometheusViews(prefix string) (views Views[Page_t], err error) {
	self := &Prometheus_t{
		Request:           prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "request"}, []string{"page", "entry"}),
		Pending:           prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "pending"}, []string{"page", "entry"}),
		Processed:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "processed"}, []string{"page", "entry", "status"}),
		Error:             prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "error"}, []string{"page", "entry", "error"}),
		LatencyMedian:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency_median"}, []string{"page", "entry"}),
		LatencyMedianSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency_median_size"}, []string{"page", "entry"}),
	}
	if err = prometheus.Register(self.Request); err != nil {
		return
	}
	if err = prometheus.Register(self.Pending); err != nil {
		return
	}
	if err = prometheus.Register(self.Processed); err != nil {
		return
	}
	if err = prometheus.Register(self.Error); err != nil {
		return
	}
	if err = prometheus.Register(self.LatencyMedian); err != nil {
		return
	}
	if err = prometheus.Register(self.LatencyMedianSize); err != nil {
		return
	}
	return self, err
}

func (self *Prometheus_t) HitBegin(ctx context.Context, page Page_t) (err error) {
	_request, err := self.Request.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_pending, err := self.Pending.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_request.Add(1)
	_pending.Add(1)
	return
}

func (self *Prometheus_t) HitEnd(ctx context.Context, page Page_t, median time.Duration, median_size int, processed int64, status string, errors string) (err error) {
	_pending, err := self.Pending.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_processed, err := self.Processed.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry, "status": status})
	if err != nil {
		return
	}
	_latency, err := self.LatencyMedian.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_latency_size, err := self.LatencyMedianSize.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_error, err := self.Error.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry, "error": errors})
	if err != nil {
		return
	}

	_pending.Add(-1)
	_processed.Add(float64(processed))
	_latency.Set(float64(median))
	_latency_size.Set(float64(median_size))
	if len(errors) > 0 {
		_error.Add(float64(processed))
	}
	return
}

func (self *Prometheus_t) HitReset(ctx context.Context, page Page_t, median time.Duration, median_size int) (err error) {
	_latency, err := self.LatencyMedian.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_latency_size, err := self.LatencyMedianSize.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}

	_latency.Set(float64(median))
	_latency_size.Set(float64(median_size))
	return
}
