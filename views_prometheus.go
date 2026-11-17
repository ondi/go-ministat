//
// sum(rate(http_page_load{app="$app_name",type="hits"}[1m])) by(page)
// sum(http_page_load{app="$app_name",type="rpm"}) by (page)
// sum(http_page_load{app="$app_name",type="pending"}) by (page)
// sum(http_page_load{app="$app_name",type="size"}) by (page)
// max(http_page_load{app="$app_name",type="max"}) by (page)
// max(http_page_load{app="$app_name",type="avg"}) by (page)
// max(http_page_load{app="$app_name",type="med"}) by (page)
// sum(rate(http_page_load{app="$app_name",type="tag"}[1m])) by (page,result)
//

package ministat

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus_t struct {
	Load *prometheus.GaugeVec // rpm, hits, pending, latency, size
}

// import "github.com/prometheus/client_golang/prometheus/promhttp"
// mux.Handle("/debug/metrics", promhttp.Handler())
func NewPrometheusViews(prefix string) (views Views[Page_t], err error) {
	self := &Prometheus_t{
		Load: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: prefix + "load"}, []string{"entry", "page", "type", "level", "tag"}),
	}
	if err = prometheus.Register(self.Load); err != nil {
		return
	}
	return self, err
}

func (self *Prometheus_t) HitCurrent(page Page_t, g []Gauge) (err error) {
	var _load prometheus.Gauge
	for _, v := range g {
		_load, err = self.Load.GetMetricWith(prometheus.Labels{
			"entry": page.Entry,
			"page":  page.Name,
			"type":  v.GetName(),
			"level": v.GetLevel(),
			"tag":   v.GetTag(),
		})
		if err != nil {
			continue
		}
		_load.Set(v.GetValueFloat64())
	}
	return
}
