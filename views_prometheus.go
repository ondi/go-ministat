//
// sum(rate(http_page_request{app="$app_name"}[1m])) by(page)
// sum(http_page_pending{app="$app_name"}) by (page)
// max(http_page_latency{app="$app_name",type="max"}) by (page)
// max(http_page_latency{app="$app_name",type="avg"}) by (page)
// max(http_page_latency{app="$app_name",type="med"}) by (page)
// sum(http_page_latency_size{app="$app_name",type="med"}) by (page)
// sum(rate(http_page_processed{app="$app_name"})[1m]) by (page,status)
// sum(rate(http_page_error{app="$app_name"}[1m])) by (page,error)
//

package ministat

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus_t struct {
	Request     *prometheus.CounterVec
	Pending     *prometheus.GaugeVec
	Gauge       *prometheus.GaugeVec
	Processed   *prometheus.CounterVec
	Error       *prometheus.CounterVec
	Latency     *prometheus.GaugeVec // label with type: avg, med, etc
	LatencySize *prometheus.GaugeVec // label with type: avg, med, etc
}

// import "github.com/prometheus/client_golang/prometheus/promhttp"
// mux.Handle("/debug/metrics", promhttp.Handler())
func NewPrometheusViews(prefix string) (views Views[Page_t], err error) {
	self := &Prometheus_t{
		Request:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "request"}, []string{"page", "entry"}),
		Pending:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "pending"}, []string{"page", "entry"}),
		Gauge:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "gauge"}, []string{"type", "page", "entry"}),
		Processed:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "processed"}, []string{"page", "entry", "status"}),
		Error:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "error"}, []string{"page", "entry", "error"}),
		Latency:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency"}, []string{"type", "page", "entry"}),
		LatencySize: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency_size"}, []string{"type", "page", "entry"}),
	}
	if err = prometheus.Register(self.Request); err != nil {
		return
	}
	if err = prometheus.Register(self.Pending); err != nil {
		return
	}
	if err = prometheus.Register(self.Gauge); err != nil {
		return
	}
	if err = prometheus.Register(self.Processed); err != nil {
		return
	}
	if err = prometheus.Register(self.Error); err != nil {
		return
	}
	if err = prometheus.Register(self.Latency); err != nil {
		return
	}
	if err = prometheus.Register(self.LatencySize); err != nil {
		return
	}
	return self, err
}

func (self *Prometheus_t) HitBegin(ctx context.Context, page Page_t, g []Gauge_t) (err error) {
	_request, err := self.Request.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_pending, err := self.Pending.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}

	var _gauge prometheus.Gauge
	for _, v := range g {
		_gauge, err = self.Gauge.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_gauge.Set(float64(v.Value))
	}

	_request.Add(1)
	_pending.Add(1)
	return
}

func (self *Prometheus_t) HitEnd(ctx context.Context, page Page_t, processed int64, status string, errors string, d []Duration_t) (err error) {
	_pending, err := self.Pending.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry})
	if err != nil {
		return
	}
	_processed, err := self.Processed.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry, "status": status})
	if err != nil {
		return
	}
	_error, err := self.Error.GetMetricWith(prometheus.Labels{"page": page.Name, "entry": page.Entry, "error": errors})
	if err != nil {
		return
	}

	var _latency, _latency_size prometheus.Gauge
	for _, v := range d {
		_latency, err = self.Latency.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_latency_size, err = self.LatencySize.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_latency.Set(float64(v.Value))
		_latency_size.Set(float64(v.Size))
	}

	_pending.Add(-1)
	_processed.Add(float64(processed))
	if len(errors) > 0 {
		_error.Add(float64(processed))
	}
	return
}

func (self *Prometheus_t) HitReset(ctx context.Context, page Page_t, g []Gauge_t, d []Duration_t) (err error) {
	var _gauge prometheus.Gauge
	for _, v := range g {
		_gauge, err = self.Gauge.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_gauge.Set(float64(v.Value))
	}
	var _latency, _latency_size prometheus.Gauge
	for _, v := range d {
		_latency, err = self.Latency.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_latency_size, err = self.LatencySize.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_latency.Set(float64(v.Value))
		_latency_size.Set(float64(v.Size))
	}
	return
}
