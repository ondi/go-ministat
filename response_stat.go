//
// RPM = sum(rate(http_request_page{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
// LATENCY = histogram_quantile(0.95, sum(rate(http_latency_page_bucket{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page, le))
// PENDING = sum(rate(http_pending_page{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
//

package ministat

import (
	"net/http"
	"strconv"
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

var pageLatency = stats.Float64(
	"http/latency/page",
	"End-to-end latency",
	stats.UnitMilliseconds,
)

var TagPageName = tag.MustNewKey("page")
var TagPageError = tag.MustNewKey("error")

var LatencyDist = view.Distribution(10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 15000, 20000, 25000, 30000)

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
		Measure:     pageLatency,
		Aggregation: LatencyDist,
	},
}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatBegin(w http.ResponseWriter, r *http.Request, name string, count int64) (*http.Request, bool) {
	r = r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))

	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, name))
	if err == nil {
		stats.Record(ctx, pagePending.M(1))
	}

	if count >= self.Count {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}

	return r, true
}

func (self *Online_t) MinistatEnd(r *http.Request, name string, status int, diff time.Duration) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, name))
	if err == nil {
		stats.Record(ctx, pagePending.M(-1))
	}

	if status > 400 && status < 500 {
		name = "/status" + strconv.FormatInt(int64(status), 10)
	}

	mutator := []tag.Mutator{
		tag.Upsert(TagPageName, name),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(TagPageError, strings.Join(v.Values(), ",")))
	}
	ctx, err = tag.New(r.Context(), mutator...)
	if err == nil {
		stats.Record(ctx, pageRequest.M(1), pageLatency.M(float64(diff.Milliseconds())))
	}
}
