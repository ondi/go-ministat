//
//
//

package ministat

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ondi/go-log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var ServerRequestCount = stats.Int64(
	"opencensus.io/http/server/page",
	"Number of HTTP requests per page",
	stats.UnitDimensionless,
)

var ServerLatency = stats.Float64(
	"opencensus.io/http/latency/page",
	"End-to-end latency",
	stats.UnitMilliseconds,
)

var TagPageName = tag.MustNewKey("page")
var TagPageError = tag.MustNewKey("error")

var latyncy_distr = view.Distribution(100, 500, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000)

var ServerRequestCountView = &view.View{
	Name:        "http/server/page",
	Description: "Count of HTTP requests per page",
	TagKeys:     []tag.Key{TagPageName, TagPageError},
	Measure:     ServerRequestCount,
	Aggregation: view.Count(),
}

var ServerRequestLatencyView = &view.View{
	Name:        "http/page/latency",
	Description: "Latency of HTTP requests per page",
	TagKeys:     []tag.Key{TagPageName},
	Measure:     ServerLatency,
	Aggregation: latyncy_distr,
}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatContext(r *http.Request) *http.Request {
	return r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))
}

func (self *Online_t) MinistatOnline(w http.ResponseWriter, r *http.Request, name string, count int64) bool {
	if count > self.Count {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return false
	}
	return true
}

func (self *Online_t) MinistatDuration(r *http.Request, name string, status int, diff time.Duration) {
	var page_name string
	switch status {
	case 401:
		page_name = "/not_authorized"
	case 404:
		page_name = "/not_found"
	default:
		page_name = name
	}
	mutator := []tag.Mutator{
		tag.Upsert(TagPageName, page_name),
	}
	if v := log.ContextGet(r.Context()); v != nil {
		mutator = append(mutator, tag.Upsert(TagPageError, strings.Join(v.Values(), ",")))
	}
	ctx, err := tag.New(r.Context(), mutator...)
	if err == nil {
		stats.Record(ctx, ServerRequestCount.M(1), ServerLatency.M(float64(diff.Milliseconds())))
	}
}
