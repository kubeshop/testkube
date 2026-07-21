package formatters

import (
	"encoding/json"
	"math"
)

// defaultInsightSeriesSamples is the default maximum number of downsampled
// points returned per segment by FormatInsightMetricSeries.
const defaultInsightSeriesSamples = 50

// --- list_insight_series -----------------------------------------------------

type insightCatalogInput struct {
	Items []struct {
		SeriesID     string                     `json:"seriesId"`
		WorkflowName string                     `json:"workflowName"`
		Source       string                     `json:"source"`
		MetricKey    string                     `json:"metricKey"`
		Identity     map[string]json.RawMessage `json:"identity"`
	} `json:"items"`
	HasMore bool `json:"hasMore"`
}

type formattedInsightSeries struct {
	SeriesID  string                     `json:"seriesId"`
	Source    string                     `json:"source,omitempty"`
	MetricKey string                     `json:"metricKey"`
	Workflow  string                     `json:"workflow,omitempty"`
	Identity  map[string]json.RawMessage `json:"identity,omitempty"`
}

// FormatInsightSeriesCatalog compacts the granular insight series catalog,
// dropping the redundant organization/environment fields (the tool is already
// environment-scoped).
func FormatInsightSeriesCatalog(raw string) (string, error) {
	input, isEmpty, err := ParseJSON[insightCatalogInput](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(input.Items) == 0 {
		return "No insight series found.", nil
	}

	series := make([]formattedInsightSeries, 0, len(input.Items))
	for _, item := range input.Items {
		series = append(series, formattedInsightSeries{
			SeriesID:  item.SeriesID,
			Source:    item.Source,
			MetricKey: item.MetricKey,
			Workflow:  item.WorkflowName,
			Identity:  item.Identity,
		})
	}

	return FormatJSON(struct {
		Series  []formattedInsightSeries `json:"series"`
		HasMore bool                     `json:"hasMore"`
	}{Series: series, HasMore: input.HasMore})
}

// --- list_insight_metric_keys ------------------------------------------------

type insightMetricKeysInput struct {
	Items   []string `json:"items"`
	HasMore bool     `json:"hasMore"`
}

// FormatInsightMetricKeys compacts the distinct granular insight metric keys.
func FormatInsightMetricKeys(raw string) (string, error) {
	input, isEmpty, err := ParseJSON[insightMetricKeysInput](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(input.Items) == 0 {
		return "No insight metric keys found.", nil
	}

	return FormatJSON(struct {
		MetricKeys []string `json:"metricKeys"`
		HasMore    bool     `json:"hasMore"`
	}{MetricKeys: input.Items, HasMore: input.HasMore})
}

// --- get_insight_metric_series -----------------------------------------------

type insightSeriesDatum struct {
	Ts       int64   `json:"ts"`
	Value    float64 `json:"value"`
	Segments []struct {
		Label string  `json:"label"`
		Value float64 `json:"value"`
	} `json:"segments"`
}

type insightSeriesPoint struct {
	Ts    int64   `json:"ts"`
	Value float64 `json:"value"`
}

type formattedInsightSegment struct {
	Segment string               `json:"segment"`
	Points  int                  `json:"points"`
	From    int64                `json:"from"`
	To      int64                `json:"to"`
	Min     float64              `json:"min"`
	Max     float64              `json:"max"`
	Avg     float64              `json:"avg"`
	Latest  float64              `json:"latest"`
	Samples []insightSeriesPoint `json:"samples"`
}

// FormatInsightMetricSeries turns a time series (TimeSeriesData) into a compact,
// per-segment summary with downsampled points. maxSamples <= 0 uses the default.
func FormatInsightMetricSeries(raw string, maxSamples int) (string, error) {
	data, isEmpty, err := ParseJSON[[]insightSeriesDatum](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(data) == 0 {
		return "No metric data found for the given query.", nil
	}
	if maxSamples <= 0 {
		maxSamples = defaultInsightSeriesSamples
	}

	// When any bucket carries segments, the response is segmented: the values
	// live in the segments and the server fills the rest of the time range with
	// empty (value 0, no segment) buckets. Drop those empty buckets so they
	// don't distort the summary. Only when no bucket has segments do we use the
	// top-level value as a single "all" series.
	segmented := false
	for _, d := range data {
		if len(d.Segments) > 0 {
			segmented = true
			break
		}
	}

	// Preserve first-seen segment order for stable output.
	order := make([]string, 0)
	points := make(map[string][]insightSeriesPoint)
	add := func(label string, ts int64, value float64) {
		if _, ok := points[label]; !ok {
			order = append(order, label)
		}
		points[label] = append(points[label], insightSeriesPoint{Ts: ts, Value: value})
	}

	total := 0
	for _, d := range data {
		if !segmented {
			add("all", d.Ts, d.Value)
			total++
			continue
		}
		for _, s := range d.Segments {
			add(s.Label, d.Ts, s.Value)
			total++
		}
	}

	segments := make([]formattedInsightSegment, 0, len(order))
	for _, label := range order {
		pts := points[label]
		if len(pts) == 0 {
			continue
		}
		min, max, sum := pts[0].Value, pts[0].Value, 0.0
		for _, p := range pts {
			if p.Value < min {
				min = p.Value
			}
			if p.Value > max {
				max = p.Value
			}
			sum += p.Value
		}
		segments = append(segments, formattedInsightSegment{
			Segment: label,
			Points:  len(pts),
			From:    pts[0].Ts,
			To:      pts[len(pts)-1].Ts,
			Min:     min,
			Max:     max,
			Avg:     sum / float64(len(pts)),
			Latest:  pts[len(pts)-1].Value,
			Samples: downsampleInsightPoints(pts, maxSamples),
		})
	}

	return FormatJSON(struct {
		PointCount int                       `json:"pointCount"`
		Series     []formattedInsightSegment `json:"series"`
	}{PointCount: total, Series: segments})
}

// downsampleInsightPoints evenly reduces a series to at most maxPoints.
// For maxPoints >= 2 it always keeps the first and last point.
func downsampleInsightPoints(values []insightSeriesPoint, maxPoints int) []insightSeriesPoint {
	if maxPoints <= 0 || len(values) <= maxPoints {
		return values
	}
	if maxPoints == 1 {
		return values[len(values)-1:]
	}

	result := make([]insightSeriesPoint, 0, maxPoints)
	result = append(result, values[0])
	step := float64(len(values)-1) / float64(maxPoints-1)
	for i := 1; i < maxPoints-1; i++ {
		idx := int(math.Round(float64(i) * step))
		result = append(result, values[idx])
	}
	result = append(result, values[len(values)-1])
	return result
}

// --- list_insight_executions -------------------------------------------------

// insightExecutionRef is one element of the insight-executions response. The
// control-plane endpoint encodes the response as a bare JSON array of these refs
// (see FormatInsightExecutions), not an object.
type insightExecutionRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Parent      string `json:"parent"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Duration    int    `json:"duration"`
	RunAt       string `json:"runAt"`
}

type formattedInsightExecution struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Workflow   string `json:"workflow,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMs int    `json:"durationMs"`
	RunAt      string `json:"runAt,omitempty"`
}

// FormatInsightExecutions projects execution references down to the fields
// useful for pivoting to other execution tools, dropping heavy report/health data.
//
// The control-plane endpoint GET /organizations/{id}/insights/series/executions
// encodes its response as a bare JSON array of execution refs. Pagination is
// carried in the Link HTTP response header, not the body, so this formatter has
// no hasMore flag to surface; callers paginate via the page/pageSize parameters.
func FormatInsightExecutions(raw string) (string, error) {
	input, isEmpty, err := ParseJSON[[]insightExecutionRef](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(input) == 0 {
		return "No executions found for the given query.", nil
	}

	executions := make([]formattedInsightExecution, 0, len(input))
	for _, e := range input {
		executions = append(executions, formattedInsightExecution{
			ID:         e.ID,
			Name:       e.Name,
			Workflow:   e.Parent,
			Status:     e.Status,
			DurationMs: e.Duration,
			RunAt:      e.RunAt,
		})
	}

	return FormatJSON(struct {
		Executions []formattedInsightExecution `json:"executions"`
	}{Executions: executions})
}
