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

	"github.com/google/uuid"
	"github.com/ondi/go-log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var TagPageName = tag.MustNewKey("page")
var TagPageError = tag.MustNewKey("error")

var pageRequest = stats.Int64(
	"http_request_count",
	"Number of HTTP requests per page",
	stats.UnitDimensionless,
)

var pagePending = stats.Int64(
	"http_pending_sum",
	"Number of HTTP pending requests per page",
	stats.UnitDimensionless,
)

var pageLatencySum = stats.Int64(
	"http_latency_sum",
	"End-to-end latency",
	stats.UnitDimensionless,
)

var pageLatencyCount = stats.Int64(
	"http_latency_num",
	"End-to-end latency",
	stats.UnitDimensionless,
)

var Views = []*view.View{
	{
		Name:        "http_request_count",
		Description: "Count of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName, TagPageError},
		Measure:     pageRequest,
		Aggregation: view.Count(),
	},
	{
		Name:        "http_pending_sum",
		Description: "Count of HTTP pending requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pagePending,
		Aggregation: view.Sum(),
	},
	{
		Name:        "http_latency_sum",
		Description: "Latency of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pageLatencySum,
		Aggregation: view.Sum(),
	},
	{
		Name:        "http_latency_num",
		Description: "Latency of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pageLatencyCount,
		Aggregation: view.Sum(),
	},
}

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

type NoOnline_t struct {
	Count int64
}

func (self *NoOnline_t) MinistatContext(w http.ResponseWriter, r *http.Request, page string, online int64) (*http.Request, bool) {
	if online >= self.Count {
		log.WarnCtx(r.Context(), "TOO MANY REQUESTS: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}
	return r, true
}

func (*NoOnline_t) MinistatBefore(r *http.Request, page string) {}

func (*NoOnline_t) MinistatAfter(r *http.Request, page string) {}

func (*NoOnline_t) MinistatDuration(r *http.Request, page string, status int, diff time.Duration) {}

func (*NoOnline_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatContext(w http.ResponseWriter, r *http.Request, page string, online int64) (*http.Request, bool) {
	r = r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))
	if online >= self.Count {
		log.WarnCtx(r.Context(), "TOO MANY REQUESTS: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}
	return r, true
}

func (self *Online_t) MinistatBefore(r *http.Request, page string) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, pagePending.M(1), pageRequest.M(1), pageLatencyCount.M(1))
	}
}

func (self *Online_t) MinistatAfter(r *http.Request, page string) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, pagePending.M(-1))
	}
}

func (self *Online_t) MinistatDuration(r *http.Request, page string, status int, diff time.Duration) {
	mutator := []tag.Mutator{
		tag.Upsert(TagPageName, page),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(TagPageError, strings.Join(v.Values(), ",")))
	}
	ctx, err := tag.New(r.Context(), mutator...)
	if err != nil {
		log.WarnCtx(r.Context(), "MINISTAT: %v", err)
	} else {
		stats.Record(ctx, pageLatencySum.M(int64(diff)))
	}
}

func (self *Online_t) MinistatEvict(page string, DurationSum time.Duration, DurationNum time.Duration) {
	ctx, err := tag.New(context.Background(), tag.Upsert(TagPageName, page))
	if err != nil {
		log.Warn("MINISTAT: %v", err)
	} else {
		stats.Record(ctx, pageLatencySum.M(-int64(DurationSum)), pageLatencyCount.M(-int64(DurationNum)))
	}
}
