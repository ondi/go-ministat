//
// RPM = sum(rate(http_request_count{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
// PENDING = sum(http_pending_sum{kubernetes_pod_name=~"POD_NAME.*"}) by (page)
// LATENCY = histogram_quantile(0.95, sum(rate(http_latency_hist_bucket{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page, le)) # DEPRECATED
// LATENCT = sum(http_latency_sum{kubernetes_pod_name=~"POD_NAME.*"}) by (page)/sum(http_latency_num{kubernetes_pod_name=~"POD_NAME.*"}) by (page)
//

package ministat

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ondi/go-log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type Views interface {
	MinistatBefore(r *http.Request, page string)
	MinistatAfter(r *http.Request, page string)
	MinistatDuration(r *http.Request, page string, status int, diff time.Duration)
	MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration)
	List() []*view.View
}

type no_views_t struct{}

func NewNoViews(prefix string) (Views, error) { return &no_views_t{}, nil }

func (*no_views_t) MinistatBefore(r *http.Request, page string) {}

func (*no_views_t) MinistatAfter(r *http.Request, page string) {}

func (*no_views_t) MinistatDuration(r *http.Request, page string, status int, diff time.Duration) {}

func (*no_views_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {}

func (*no_views_t) List() []*view.View { return nil }

type views_t struct {
	pageName       tag.Key
	pageError      tag.Key
	pageRequest    *stats.Int64Measure
	pagePending    *stats.Int64Measure
	pageLatencySum *stats.Int64Measure
	pageLatencyNum *stats.Int64Measure
	views          []*view.View
}

func NewViews(prefix string) (Views, error) {
	self := &views_t{
		pageRequest:    stats.Int64("request_count", "number of requests", stats.UnitDimensionless),
		pagePending:    stats.Int64("pending_sum", "number of pending requests", stats.UnitDimensionless),
		pageLatencySum: stats.Int64("latency_sum", "latency numerator", stats.UnitDimensionless),
		pageLatencyNum: stats.Int64("latency_num", "latency denominator", stats.UnitDimensionless),
	}
	var err error
	if self.pageName, err = tag.NewKey("page"); err != nil {
		return nil, err
	}
	if self.pageError, err = tag.NewKey("error"); err != nil {
		return nil, err
	}
	self.views = []*view.View{
		{
			Name:        prefix + "request_count",
			Description: "count of requests",
			TagKeys:     []tag.Key{self.pageName, self.pageError},
			Measure:     self.pageRequest,
			Aggregation: view.Count(),
		},
		{
			Name:        prefix + "pending_sum",
			Description: "count of pending requests",
			TagKeys:     []tag.Key{self.pageName},
			Measure:     self.pagePending,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "latency_sum",
			Description: "latency numerator",
			TagKeys:     []tag.Key{self.pageName},
			Measure:     self.pageLatencySum,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "latency_num",
			Description: "latency denominator",
			TagKeys:     []tag.Key{self.pageName},
			Measure:     self.pageLatencyNum,
			Aggregation: view.Sum(),
		},
	}
	return self, err
}

func (self *views_t) List() []*view.View {
	return self.views
}

func (self *views_t) MinistatBefore(r *http.Request, page string) {
	ctx, err := tag.New(r.Context(), tag.Upsert(self.pageName, page))
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, self.pagePending.M(1), self.pageRequest.M(1), self.pageLatencyNum.M(1))
	}
}

func (self *views_t) MinistatAfter(r *http.Request, page string) {
	ctx, err := tag.New(r.Context(), tag.Upsert(self.pageName, page))
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, self.pagePending.M(-1))
	}
}

func (self *views_t) MinistatDuration(r *http.Request, page string, status int, diff time.Duration) {
	mutator := []tag.Mutator{
		tag.Upsert(self.pageName, page),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(self.pageError, strings.Join(v.Values(), ",")))
	}
	ctx, err := tag.New(r.Context(), mutator...)
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, self.pageLatencySum.M(int64(diff)))
	}
}

func (self *views_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {
	ctx, err := tag.New(context.Background(), tag.Upsert(self.pageName, page))
	if err != nil {
		log.Warn("MINISTAT: %v", err)
	} else {
		stats.Record(ctx, self.pageLatencySum.M(-int64(DurationSum)), self.pageLatencyNum.M(-int64(DurationNum)))
	}
}
