//
// sum(rate(http_page_load{app="$app_name",type="hits"}[1m])) by(page)
// sum(http_page_load{app="$app_name",type="pending"}) by (page)
// sum(http_page_load{app="$app_name",type="rps"}) by (page)
// sum(http_page_load{app="$app_name",type="size"}) by (page)
// max(http_page_latency{app="$app_name",type="max"}) by (page)
// max(http_page_latency{app="$app_name",type="avg"}) by (page)
// max(http_page_latency{app="$app_name",type="med"}) by (page)
// sum(rate(http_page_processed{app="$app_name"})[1m]) by (page,status)
// sum(rate(http_page_error{app="$app_name"}[1m])) by (page,error)
//

package ministat

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus_t struct {
	Load      *prometheus.GaugeVec // pending, hits, rps, size
	Latency   *prometheus.GaugeVec // med, max, avg
	Processed *prometheus.CounterVec
	Error     *prometheus.CounterVec
}

// import "github.com/prometheus/client_golang/prometheus/promhttp"
// mux.Handle("/debug/metrics", promhttp.Handler())
func NewPrometheusViews(prefix string) (views Views[Page_t], err error) {
	self := &Prometheus_t{
		Load:      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "load"}, []string{"type", "page", "entry"}),
		Latency:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "latency"}, []string{"type", "page", "entry"}),
		Processed: prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "processed"}, []string{"page", "entry", "status"}),
		Error:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + "error"}, []string{"page", "entry", "error"}),
	}
	if err = prometheus.Register(self.Load); err != nil {
		return
	}
	if err = prometheus.Register(self.Latency); err != nil {
		return
	}
	if err = prometheus.Register(self.Processed); err != nil {
		return
	}
	if err = prometheus.Register(self.Error); err != nil {
		return
	}
	return self, err
}

func (self *Prometheus_t) HitBegin(ctx context.Context, page Page_t, g []Gauge_t) (err error) {
	var _load prometheus.Gauge
	for _, v := range g {
		_load, err = self.Load.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_load.Set(float64(v.Value))
	}
	return
}

func (self *Prometheus_t) HitEnd(ctx context.Context, page Page_t, processed int64, status string, errors string, g []Gauge_t, d []Duration_t) (err error) {
	if err = self.HitRefresh(ctx, page, g, d); err != nil {
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

	_processed.Add(float64(processed))
	if len(errors) > 0 {
		_error.Add(float64(processed))
	}
	return
}

func (self *Prometheus_t) HitRefresh(ctx context.Context, page Page_t, g []Gauge_t, d []Duration_t) (err error) {
	var _load, _latency prometheus.Gauge
	for _, v := range g {
		_load, err = self.Load.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_load.Set(float64(v.Value))
	}
	for _, v := range d {
		_latency, err = self.Latency.GetMetricWith(prometheus.Labels{"type": v.Label, "page": page.Name, "entry": page.Entry})
		if err != nil {
			return
		}
		_latency.Set(float64(v.Value))
	}
	return
}
