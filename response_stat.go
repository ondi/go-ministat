//
// RPM = sum(rate(http_request_page{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page)
// LATENCY = histogram_quantile(0.95, sum(rate(http_latency_page_bucket{kubernetes_pod_name=~"POD_NAME.*"}[1m])) by(page, le))
// PENDING = http_pending_page{kubernetes_pod_name=~"POD_NAME.*"}
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

var pageLatencyDist = stats.Float64(
	"http/latency/page",
	"End-to-end latency",
	stats.UnitMilliseconds,
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
}

func GetPageName(r *http.Request) (res string) {
	return r.URL.Path
}

type NoOnline_t struct{}

func (NoOnline_t) MinistatBegin(w http.ResponseWriter, r *http.Request, name string, count int64) (*http.Request, bool) {
	return r, true
}

func (NoOnline_t) MinistatEnd(r *http.Request, name string, status int, diff time.Duration) {
	return
}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatBegin(w http.ResponseWriter, r *http.Request, page string, online int64) (*http.Request, bool) {
	r = r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))

	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err == nil {
		stats.Record(ctx, pagePending.M(1))
	}

	if online >= self.Count {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return r, false
	}

	return r, true
}

func (self *Online_t) MinistatEnd(r *http.Request, page string, status int, diff time.Duration) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, page))
	if err == nil {
		stats.Record(ctx, pagePending.M(-1))
	}

	switch status {
	case 401, 404:
		page = "/page_" + strconv.FormatInt(int64(status), 10)
	}

	mutator := []tag.Mutator{
		tag.Upsert(TagPageName, page),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(TagPageError, strings.Join(v.Values(), ",")))
	}
	ctx, err = tag.New(r.Context(), mutator...)
	if err == nil {
		stats.Record(ctx, pageRequest.M(1), pageLatencyDist.M(float64(diff)/1e6))
	}
}
