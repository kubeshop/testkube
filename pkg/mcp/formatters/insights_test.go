package formatters

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatInsightSeriesCatalog(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out, err := FormatInsightSeriesCatalog(`{"items":[],"hasMore":false}`)
		require.NoError(t, err)
		assert.Equal(t, "No insight series found.", out)
	})

	t.Run("drops org/env and keeps identity", func(t *testing.T) {
		raw := `{"items":[{"seriesId":"s1","organizationId":"org","environmentId":"env","workflowName":"wf","source":"k6","metricKey":"http_req_duration_p95_ms","identity":{"route":"/api"}}],"hasMore":true}`
		out, err := FormatInsightSeriesCatalog(raw)
		require.NoError(t, err)
		assert.NotContains(t, out, "organizationId")
		assert.NotContains(t, out, "environmentId")
		assert.Contains(t, out, "http_req_duration_p95_ms")
		assert.Contains(t, out, `"route":"/api"`)
		assert.Contains(t, out, `"hasMore":true`)
	})
}

func TestFormatInsightMetricKeys(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out, err := FormatInsightMetricKeys(`{"items":[],"hasMore":false}`)
		require.NoError(t, err)
		assert.Equal(t, "No insight metric keys found.", out)
	})

	t.Run("lists keys", func(t *testing.T) {
		out, err := FormatInsightMetricKeys(`{"items":["a","b"],"hasMore":false}`)
		require.NoError(t, err)
		assert.Contains(t, out, `"metricKeys":["a","b"]`)
	})
}

func TestFormatInsightMetricSeries(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out, err := FormatInsightMetricSeries(`[]`, 0)
		require.NoError(t, err)
		assert.Contains(t, out, "No metric data found")
	})

	t.Run("non-segmented computes summary", func(t *testing.T) {
		raw := `[{"ts":1,"value":10,"segments":[]},{"ts":2,"value":30,"segments":[]},{"ts":3,"value":20,"segments":[]}]`
		out, err := FormatInsightMetricSeries(raw, 0)
		require.NoError(t, err)

		var parsed struct {
			PointCount int `json:"pointCount"`
			Series     []struct {
				Segment string  `json:"segment"`
				Points  int     `json:"points"`
				Min     float64 `json:"min"`
				Max     float64 `json:"max"`
				Avg     float64 `json:"avg"`
				Latest  float64 `json:"latest"`
			} `json:"series"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		require.Len(t, parsed.Series, 1)
		assert.Equal(t, 3, parsed.PointCount)
		assert.Equal(t, "all", parsed.Series[0].Segment)
		assert.Equal(t, float64(10), parsed.Series[0].Min)
		assert.Equal(t, float64(30), parsed.Series[0].Max)
		assert.Equal(t, float64(20), parsed.Series[0].Avg)
		assert.Equal(t, float64(20), parsed.Series[0].Latest)
	})

	t.Run("splits per segment", func(t *testing.T) {
		raw := `[{"ts":1,"value":0,"segments":[{"label":"passed","value":5},{"label":"failed","value":1}]}]`
		out, err := FormatInsightMetricSeries(raw, 0)
		require.NoError(t, err)
		assert.Contains(t, out, `"segment":"passed"`)
		assert.Contains(t, out, `"segment":"failed"`)
	})

	t.Run("drops empty generate_series buckets when segmented", func(t *testing.T) {
		// The server pads the time range with empty (value 0, no segment)
		// buckets around the real data buckets. These must not become an "all"
		// series of zeros that distorts min/avg.
		raw := `[
			{"ts":1,"value":0,"segments":[]},
			{"ts":2,"value":180,"segments":[{"label":"http_req_duration_p95_ms","value":180}]},
			{"ts":3,"value":0,"segments":[]},
			{"ts":4,"value":240,"segments":[{"label":"http_req_duration_p95_ms","value":240}]}
		]`
		out, err := FormatInsightMetricSeries(raw, 0)
		require.NoError(t, err)
		assert.NotContains(t, out, `"segment":"all"`)

		var parsed struct {
			PointCount int `json:"pointCount"`
			Series     []struct {
				Segment string  `json:"segment"`
				Points  int     `json:"points"`
				Min     float64 `json:"min"`
				Max     float64 `json:"max"`
			} `json:"series"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		require.Len(t, parsed.Series, 1)
		assert.Equal(t, 2, parsed.PointCount)
		assert.Equal(t, "http_req_duration_p95_ms", parsed.Series[0].Segment)
		assert.Equal(t, float64(180), parsed.Series[0].Min)
		assert.Equal(t, float64(240), parsed.Series[0].Max)
	})

	t.Run("downsamples to maxSamples keeping ends", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("[")
		for i := 0; i < 100; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"ts":`)
			sb.WriteString(itoa(i))
			sb.WriteString(`,"value":`)
			sb.WriteString(itoa(i))
			sb.WriteString(`,"segments":[]}`)
		}
		sb.WriteString("]")

		out, err := FormatInsightMetricSeries(sb.String(), 10)
		require.NoError(t, err)
		var parsed struct {
			Series []struct {
				Points  int `json:"points"`
				Samples []struct {
					Ts int64 `json:"ts"`
				} `json:"samples"`
			} `json:"series"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		require.Len(t, parsed.Series, 1)
		assert.Equal(t, 100, parsed.Series[0].Points)
		assert.Len(t, parsed.Series[0].Samples, 10)
		assert.Equal(t, int64(0), parsed.Series[0].Samples[0].Ts)
		assert.Equal(t, int64(99), parsed.Series[0].Samples[9].Ts)
	})
}

func TestFormatInsightExecutions(t *testing.T) {
	// The control-plane endpoint encodes its response as a bare JSON array of
	// execution refs (pagination lives in the Link header, not the body), so the
	// formatter's inputs mirror that shape.
	t.Run("empty", func(t *testing.T) {
		out, err := FormatInsightExecutions(`[]`)
		require.NoError(t, err)
		assert.Contains(t, out, "No executions found")
	})

	t.Run("null", func(t *testing.T) {
		out, err := FormatInsightExecutions(`null`)
		require.NoError(t, err)
		assert.Contains(t, out, "No executions found")
	})

	t.Run("projects fields and drops heavy data", func(t *testing.T) {
		raw := `[{"id":"e1","name":"n1","parent":"wf","environment":"env","status":"passed","duration":1200,"runAt":"2026-07-01T00:00:00Z","reports":[{"ref":"big"}],"health":{"x":1}}]`
		out, err := FormatInsightExecutions(raw)
		require.NoError(t, err)
		assert.Contains(t, out, `"id":"e1"`)
		assert.Contains(t, out, `"workflow":"wf"`)
		assert.Contains(t, out, `"durationMs":1200`)
		assert.NotContains(t, out, "reports")
		assert.NotContains(t, out, "health")
		assert.NotContains(t, out, "environment")
	})

	t.Run("rejects the legacy object shape", func(t *testing.T) {
		// Guards against regressing to the object shape: the real endpoint never
		// returns this, and accepting it silently would re-hide the array mismatch.
		_, err := FormatInsightExecutions(`{"executions":[],"hasMore":false}`)
		require.Error(t, err)
	})
}

// itoa is a tiny helper to avoid importing strconv in the test builder above.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
