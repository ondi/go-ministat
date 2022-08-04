//
// RPM = sum(rate(http_request_count{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
// PENDING = sum(http_pending_sum{kubernetes_pod_name=~"POD_NAME.*"}) by (page)
// LATENCY = histogram_quantile(0.95, sum(rate(http_latency_hist_bucket{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page, le)) # DEPRECATED
// LATENCT = sum(http_latency_sum{kubernetes_pod_name=~"POD_NAME.*"}) by (page)/sum(http_latency_num{kubernetes_pod_name=~"POD_NAME.*"}) by (page)
//

package ministat

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ondi/go-log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type Evict interface {
	MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration)
}

type Views interface {
	Evict
	MinistatBefore(ctx context.Context, page string)
	MinistatAfter(ctx context.Context, page string)
	MinistatDuration(ctx context.Context, page string, diff time.Duration, processed int64, status int)
	List() []*view.View
}

type no_views_t struct{}

func NewNoViews(prefix string) (Views, error) { return &no_views_t{}, nil }

func (*no_views_t) MinistatBefore(ctx context.Context, page string) {}

func (*no_views_t) MinistatAfter(ctx context.Context, page string) {}

func (*no_views_t) MinistatDuration(ctx context.Context, page string, diff time.Duration, processed int64, status int) {
}

func (*no_views_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {
}

func (*no_views_t) List() []*view.View { return nil }

type views_t struct {
	pageName       tag.Key
	pageError      tag.Key
	pageStatus     tag.Key
	pageRequest    *stats.Int64Measure
	pagePayload    *stats.Int64Measure
	pagePending    *stats.Int64Measure
	pageLatencySum *stats.Int64Measure
	pageLatencyNum *stats.Int64Measure
	views          []*view.View
}

func NewViews(prefix string) (Views, error) {
	self := &views_t{
		pageRequest:    stats.Int64("request_count", "number of requests", stats.UnitDimensionless),
		pagePayload:    stats.Int64("payload_count", "number of payload processed", stats.UnitDimensionless),
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
	if self.pageStatus, err = tag.NewKey("status"); err != nil {
		return nil, err
	}
	self.views = []*view.View{
		{
			Name:        prefix + "request_count",
			Description: "number of requests",
			TagKeys:     []tag.Key{self.pageName},
			Measure:     self.pageRequest,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "payload_count",
			Description: "number of payload processed",
			TagKeys:     []tag.Key{self.pageName, self.pageStatus, self.pageError},
			Measure:     self.pagePayload,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "pending_sum",
			Description: "number of pending requests",
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

func (self *views_t) MinistatBefore(ctx context.Context, page string) {
	ctx, err := tag.New(ctx, tag.Upsert(self.pageName, page))
	if err != nil {
		log.WarnCtx(ctx, "MINISTAT: %v %q", err, page)
	} else {
		stats.Record(ctx, self.pagePending.M(1), self.pageRequest.M(1), self.pageLatencyNum.M(1))
	}
}

func (self *views_t) MinistatAfter(ctx context.Context, page string) {
	ctx, err := tag.New(ctx, tag.Upsert(self.pageName, page))
	if err != nil {
		log.WarnCtx(ctx, "MINISTAT: %v %q", err, page)
	} else {
		stats.Record(ctx, self.pagePending.M(-1))
	}
}

func (self *views_t) MinistatDuration(ctx context.Context, page string, diff time.Duration, processed int64, status int) {
	mutator := []tag.Mutator{
		tag.Upsert(self.pageName, page),
		tag.Upsert(self.pageStatus, strconv.FormatInt(int64(status), 10)),
	}
	if v := log.ContextGet(ctx); v != nil {
		mutator = append(mutator, tag.Upsert(self.pageError, strings.Join(v.Values(), ",")))
	}
	ctx, err := tag.New(ctx, mutator...)
	if err != nil {
		log.WarnCtx(ctx, "MINISTAT: %v %q", err, page)
	} else {
		stats.Record(ctx, self.pageLatencySum.M(int64(diff)), self.pagePayload.M(processed))
	}
}

func (self *views_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {
	ctx, err := tag.New(context.Background(), tag.Upsert(self.pageName, page))
	if err != nil {
		log.Warn("MINISTAT: %v %q", err, page)
	} else {
		stats.Record(ctx, self.pageLatencySum.M(-int64(DurationSum)), self.pageLatencyNum.M(-int64(DurationNum)))
	}
}
