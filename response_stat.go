//
// RPM = sum(rate(http_request_page{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
// LATENCY = histogram_quantile(0.95, sum(rate(http_latency_page_bucket{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page, le))
// PENDING = http_pending_page{kubernetes_pod_name=~"POD_NAME.*"}
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

var pageRequest = stats.Int64(
	"http/request/page",
	"Number of HTTP requests per page",
	stats.UnitDimensionless,
)

var pagePending = stats.Int64(
	"http/pending/page",
	"Number of HTTP pending requests per page",
	stats.UnitDimensionless,
)

var pageLatencyDist = stats.Float64(
	"http/latency/page",
	"End-to-end latency",
	stats.UnitMilliseconds,
)

var pageLatencySum = stats.Int64(
	"http/latency_sum",
	"End-to-end latency",
	stats.UnitDimensionless,
)

var pageLatencyCount = stats.Int64(
	"http/latency_count",
	"End-to-end latency",
	stats.UnitDimensionless,
)

var TagPageName = tag.MustNewKey("page")
var TagPageError = tag.MustNewKey("error")

var LatencyDist = view.Distribution(50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000)

var Views = []*view.View{
	{
		Name:        "http/request/page",
		Description: "Count of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName, TagPageError},
		Measure:     pageRequest,
		Aggregation: view.Count(),
	},
	{
		Name:        "http/pending/page",
		Description: "Count of HTTP pending requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pagePending,
		Aggregation: view.Sum(),
	},
	{
		Name:        "http/latency/page",
		Description: "Latency of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pageLatencyDist,
		Aggregation: LatencyDist,
	},
	{
		Name:        "http/latency_sum",
		Description: "Latency of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     pageLatencySum,
		Aggregation: view.Sum(),
	},
	{
		Name:        "http/latency_count",
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
		log.ErrorCtx(r.Context(), "TOO MANY REQUEST: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}
	return r, true
}

func (*NoOnline_t) MinistatBegin(r *http.Request, page string) {

}

func (*NoOnline_t) MinistatEnd(r *http.Request, page string, status int, diff time.Duration) {

}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatContext(w http.ResponseWriter, r *http.Request, page string, online int64) (*http.Request, bool) {
	r = r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))
	if online >= self.Count {
		log.ErrorCtx(r.Context(), "TOO MANY REQUEST: %v", page)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}
	return r, true
}

func (self *Online_t) MinistatBegin(r *http.Request, page string) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err == nil {
		stats.Record(ctx, pagePending.M(1))
	}
}

func (self *Online_t) MinistatEnd(r *http.Request, page string, status int, diff time.Duration) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err == nil {
		stats.Record(ctx, pagePending.M(-1))
	}

	mutator := []tag.Mutator{
		tag.Upsert(TagPageName, page),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(TagPageError, strings.Join(v.Values(), ",")))
	}
	ctx, err = tag.New(r.Context(), mutator...)
	if err == nil {
		stats.Record(ctx, pageRequest.M(1), pageLatencyDist.M(float64(diff)/1e6), pageLatencySum.M(int64(diff)), pageLatencyCount.M(1))
	}
}

func Evict(f func(f func(page string, value *Counter_t) bool)) {
	f(
		func(page string, value *Counter_t) bool {
			mutator := []tag.Mutator{
				tag.Upsert(TagPageName, page),
			}
			ctx, err := tag.New(context.Background(), mutator...)
			if err == nil {
				stats.Record(ctx, pageLatencySum.M(-int64(value.DurationSum)), pageLatencyCount.M(-int64(value.DurationNum)))
			}
			return true
		},
	)
}
