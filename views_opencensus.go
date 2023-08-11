//
// RPS = sum(rate(http_page_request{app="$app_name"}[1m])) by(page)
// PENDING = sum(http_page_pending{app="$app_name"}) by (page)
// LATENCY = avg(http_page_latency_median{app="$app_name"}) by (page)
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

type OpencensusViews_t struct {
	tagPage               tag.Key
	tagError              tag.Key
	tagStatus             tag.Key
	pageRequest           *stats.Int64Measure
	pagePending           *stats.Int64Measure
	pageProcessed         *stats.Int64Measure
	pageError             *stats.Int64Measure
	pageLatencyMedian     *stats.Int64Measure
	pageLatencyMedianSize *stats.Int64Measure
}

// import "contrib.go.opencensus.io/exporter/prometheus"
//
//	if exporter, err = prometheus.NewExporter(prometheus.Options{}); err == nil {
//		oc.Handle("/debug/metrics", exporter)
//	}
func NewOpencensusViews(prefix string) (self *OpencensusViews_t, err error) {
	self = &OpencensusViews_t{
		pageRequest:           stats.Int64(prefix+"page_request", "number of requests", stats.UnitDimensionless),
		pagePending:           stats.Int64(prefix+"page_pending", "number of pending requests", stats.UnitDimensionless),
		pageProcessed:         stats.Int64(prefix+"page_processed", "processed by page", stats.UnitDimensionless),
		pageError:             stats.Int64(prefix+"page_error", "error by page", stats.UnitDimensionless),
		pageLatencyMedian:     stats.Int64(prefix+"page_latency_median", "latency median", stats.UnitDimensionless),
		pageLatencyMedianSize: stats.Int64(prefix+"page_latency_median_size", "latency median size", stats.UnitDimensionless),
	}
	if self.tagPage, err = tag.NewKey("page"); err != nil {
		return
	}
	if self.tagError, err = tag.NewKey("error"); err != nil {
		return
	}
	if self.tagStatus, err = tag.NewKey("status"); err != nil {
		return
	}
	views := []*view.View{
		{
			Name:        prefix + "page_request",
			Description: "number of requests",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageRequest,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "page_pending",
			Description: "number of pending requests",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pagePending,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "page_processed",
			Description: "processed by page",
			TagKeys:     []tag.Key{self.tagPage, self.tagStatus},
			Measure:     self.pageProcessed,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "page_error",
			Description: "error by page",
			TagKeys:     []tag.Key{self.tagPage, self.tagError},
			Measure:     self.pageError,
			Aggregation: view.Sum(),
		},
		{
			Name:        prefix + "page_latency_median",
			Description: "latency median",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageLatencyMedian,
			Aggregation: view.LastValue(),
		},
		{
			Name:        prefix + "page_latency_median_size",
			Description: "latency median size",
			TagKeys:     []tag.Key{self.tagPage},
			Measure:     self.pageLatencyMedianSize,
			Aggregation: view.LastValue(),
		},
	}
	err = view.Register(views...)
	return
}

func (self *OpencensusViews_t) HitBegin(ctx context.Context, page string) (err error) {
	var name strings.Builder
	ctx, err = tag.New(ctx,
		tag.Upsert(self.tagPage, PrintableAscii(page, &name, 255).String()),
	)
	if err != nil {
		return
	}
	stats.Record(ctx, self.pageRequest.M(1), self.pagePending.M(1))
	return
}

func (self *OpencensusViews_t) HitEnd(ctx context.Context, page string, median time.Duration, median_size int, processed int64, status int, errors string) (err error) {
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
		self.pagePending.M(-1),
		self.pageProcessed.M(processed),
		self.pageLatencyMedian.M(int64(median)),
		self.pageLatencyMedianSize.M(int64(median_size)),
	}
	if errs.Len() > 0 {
		measure = append(measure, self.pageError.M(processed))
	}
	stats.Record(ctx, measure...)
	return
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

type NoViews_t struct{}

func NewNoViews(prefix string) (*NoViews_t, error) {
	return &NoViews_t{}, nil
}

func (*NoViews_t) HitBegin(ctx context.Context, page string) (err error) {
	return
}

func (*NoViews_t) HitEnd(ctx context.Context, page string, median time.Duration, median_size int, processed int64, status int, errors string) (err error) {
	return
}
