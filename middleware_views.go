//
// RPS = sum(rate(http_request_count{app="$app_name"}[1m])) by(page)
// PENDING = sum(http_pending_sum{app="$app_name"}) by (page)
// LATENCY = avg(http_latency_median{app="$app_name"}) by (page)
//

package ministat

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type OpenCensus interface {
	OpenCensusViews() []*view.View
}

func PrintableAscii(s string, out *strings.Builder, limit int) *strings.Builder {
	for _, r := range s {
		if r >= 0x20 && r <= 0x7e {
			out.WriteRune(r)
			if out.Len() >= limit {
				return out
			}
		}
	}
	return out
}

type no_views_t struct{}

func NewNoViews(prefix string) (Views, error) { return &no_views_t{}, nil }

func (*no_views_t) HitBegin(ctx context.Context, page string) (err error) {
	return
}

func (*no_views_t) HitEnd(ctx context.Context, page string) (err error) {
	return
}

func (*no_views_t) HitDuration(ctx context.Context, page string, median time.Duration, median_size int, processed int64, status int, errors string) (err error) {
	return
}

func (*no_views_t) OpenCensusViews() []*view.View { return nil }

type views_t struct {
	tagPage               tag.Key
	tagError              tag.Key
	tagStatus             tag.Key
	pageRequest           *stats.Int64Measure
	pagePending           *stats.Int64Measure
	pageStatus            *stats.Int64Measure
	pageError             *stats.Int64Measure
	pageLatencyMedian     *stats.Int64Measure
	pageLatencyMedianSize *stats.Int64Measure
	views                 []*view.View
}

func NewViews(prefix string) (Views, error) {
	self := &views_t{
		pageRequest:           stats.Int64(prefix+"request_count", "number of requests", stats.UnitDimensionless),
		pagePending:           stats.Int64(prefix+"pending_sum", "number of pending requests", stats.UnitDimensionless),
		pageStatus:            stats.Int64(prefix+"payload_status", "status by page", stats.UnitDimensionless),
		pageError:             stats.Int64(prefix+"payload_error", "error by page", stats.UnitDimensionless),
		pageLatencyMedian:     stats.Int64(prefix+"latency_median", "latency median", stats.UnitDimensionless),
		pageLatencyMedianSize: stats.Int64(prefix+"latency_median_size", "latency median size", stats.UnitDimensionless),
	}
	var err error
	if self.tagPage, err = tag.NewKey("page"); err != nil {
		return nil, err
	}
	if self.tagError, err = tag.NewKey("error"); err != nil {
		return nil, err
	}
	if self.tagStatus, err = tag.NewKey("status"); err != nil {
		return nil, err
	}
	self.views = []*view.View{
		{
			Name:        prefix + "request_count",
			Description: "number of requests",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageRequest,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "pending_sum",
			Description: "number of pending requests",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pagePending,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "payload_status",
			Description: "payload by page",
			TagKeys:     []tag.Key{self.tagPage, self.tagStatus},
			Measure:     self.pageStatus,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "payload_error",
			Description: "error by page",
			TagKeys:     []tag.Key{self.tagPage, self.tagError},
			Measure:     self.pageError,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "latency_median",
			Description: "latency median",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageLatencyMedian,
			Aggregation: view.LastValue(),
		},
		{
			Name:        prefix + "latency_median_size",
			Description: "latency median size",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageLatencyMedianSize,
			Aggregation: view.LastValue(),
		},
	}
	return self, err
}

func (self *views_t) OpenCensusViews() []*view.View {
	return self.views
}

func (self *views_t) HitBegin(ctx context.Context, page string) (err error) {
	var sb strings.Builder
	ctx, err = tag.New(ctx,
		tag.Upsert(self.tagPage, PrintableAscii(page, &sb, 255).String()),
	)
	if err != nil {
		return
	}
	stats.Record(ctx, self.pagePending.M(1), self.pageRequest.M(1))
	return
}

func (self *views_t) HitEnd(ctx context.Context, page string) (err error) {
	var sb strings.Builder
	ctx, err = tag.New(ctx,
		tag.Upsert(self.tagPage, PrintableAscii(page, &sb, 255).String()),
	)
	if err != nil {
		return
	}
	stats.Record(ctx, self.pagePending.M(-1))
	return
}

func (self *views_t) HitDuration(ctx context.Context, page string, median time.Duration, median_size int, processed int64, status int, errors string) (err error) {
	var name, errs strings.Builder
	ctx, err = tag.New(ctx,
		tag.Upsert(self.tagPage, PrintableAscii(page, &name, 255).String()),
		tag.Upsert(self.tagError, PrintableAscii(errors, &errs, 255).String()),
		tag.Upsert(self.tagStatus, strconv.FormatInt(int64(status), 10)),
	)
	if err != nil {
		return
	}
	measure := []stats.Measurement{
		self.pageStatus.M(processed),
		self.pageLatencyMedian.M(int64(median)),
		self.pageLatencyMedianSize.M(int64(median_size)),
	}
	if errs.Len() > 0 {
		measure = append(measure, self.pageError.M(processed))
	}
	stats.Record(ctx, measure...)
	return
}
