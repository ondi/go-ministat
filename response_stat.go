//
//
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

var PageRequest = stats.Int64(
	"http/request/page",
	"Number of HTTP requests per page",
	stats.UnitDimensionless,
)

var PagePending = stats.Int64(
	"http/pending/page",
	"Number of HTTP pending requests per page",
	stats.UnitDimensionless,
)

var PageLatency = stats.Float64(
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
		Measure:     PageRequest,
		Aggregation: view.Count(),
	},
	{
		Name:        "http/pending/page",
		Description: "Count of HTTP pending requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     PagePending,
		Aggregation: view.Sum(),
	},
	{
		Name:        "http/latency/page",
		Description: "Latency of HTTP requests per page",
		TagKeys:     []tag.Key{TagPageName},
		Measure:     PageLatency,
		Aggregation: LatencyDist,
	},
}

type Online_t struct {
	Count int64
}

func (self *Online_t) MinistatContext(r *http.Request) *http.Request {
	return r.WithContext(log.ContextSet(r.Context(), log.ContextNew(uuid.New().String())))
}

func (self *Online_t) MinistatOnline(w http.ResponseWriter, r *http.Request, name string, count int64) bool {
	if count >= self.Count {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return false
	}
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, name))
	if err == nil {
		stats.Record(ctx, PagePending.M(1))
	}
	return true
}

func (self *Online_t) MinistatDuration(r *http.Request, name string, status int, diff time.Duration) {
	ctx, err := tag.New(r.Context(), tag.Upsert(TagPageName, name))
	if err == nil {
		stats.Record(ctx, PagePending.M(-1))
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
		stats.Record(ctx, PageRequest.M(1), PageLatency.M(float64(diff.Milliseconds())))
	}
}