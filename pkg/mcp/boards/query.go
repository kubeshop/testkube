package boards

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Endpoint is the org-scoped insight endpoint that renders a report kind,
// relative to /organizations/{id}.
type Endpoint string

const (
	EndpointStats      Endpoint = "/insights/stats"
	EndpointExecutions Endpoint = "/insights/executions"
	EndpointWorkflows  Endpoint = "/insights/workflows"
	EndpointSeries     Endpoint = "/insights/series"
)

// durationMinutes mirrors the dashboard's DURATION_MAP. A quarter is 4 x 30
// days there, not 90, and the port keeps it that way so both sides agree.
var durationMinutes = map[string]int{
	"day":     24 * 60,
	"week":    24 * 60 * 7,
	"month":   24 * 60 * 30,
	"quarter": 24 * 60 * 30 * 4,
}

// InsightQuery is the request that renders one report. Empty fields are not
// sent. Env comes from the report's environment filter; a report without one
// covers the whole organization, which is what the dashboard shows for it.
type InsightQuery struct {
	Endpoint  Endpoint
	StartDate time.Time
	EndDate   time.Time

	Env       string
	Workflow  string
	Status    string
	Selector  string
	TagFilter string

	// executions
	GroupBy string

	// time-series
	Measure         string
	Aggregate       string
	Segment         string
	IdentityFilters string

	// CurrentEnvironment asks the client to scope the query to the environment
	// the MCP session runs in, replacing the report's own environment filter.
	// Only the client knows that environment, so it applies it with
	// WithEnvironment before sending the query.
	CurrentEnvironment bool
}

// WithEnvironment returns the query scoped to env when it asks for the current
// environment, and unchanged otherwise.
func (q InsightQuery) WithEnvironment(env string) InsightQuery {
	if q.CurrentEnvironment {
		q.Env = env
	}
	return q
}

// QueryParams encodes the query as URL query parameters. Empty values are
// omitted.
func (q InsightQuery) QueryParams() map[string]string {
	params := map[string]string{
		"startDate": q.StartDate.UTC().Format(time.RFC3339),
		"endDate":   q.EndDate.UTC().Format(time.RFC3339),
	}
	set := func(key, value string) {
		if value != "" {
			params[key] = value
		}
	}
	set("env", q.Env)
	set("workflow", q.Workflow)
	set("status", q.Status)
	set("selector", q.Selector)
	set("tagFilter", q.TagFilter)
	set("groupBy", q.GroupBy)
	set("measure", q.Measure)
	set("aggregate", q.Aggregate)
	set("segment", q.Segment)
	set("identityFilters", q.IdentityFilters)
	return params
}

// QueryOptions controls how a report is turned into a query.
type QueryOptions struct {
	// Now anchors relative durations. Zero means time.Now().
	Now time.Time
	// CurrentEnvironment replaces the report's own environment filter with the
	// MCP session's environment (see InsightQuery.CurrentEnvironment).
	CurrentEnvironment bool
	// Location is the time zone relative durations are anchored in. The
	// dashboard anchors them to the viewer's local midnight, so this must be
	// the viewer's zone for the numbers to match. Nil means UTC.
	Location *time.Location
}

// BuildQuery translates a report into the insight query the dashboard issues
// to render it (a port of mapBaseFilterToQueryParams, useInsightsBaseFilter and
// useTimeSeriesBaseFilter).
func BuildQuery(kind string, params map[string]any, opts QueryOptions) (InsightQuery, error) {
	if !IsKind(kind) {
		return InsightQuery{}, unknownKindError(kind)
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	filters, err := ParseFilters(params["filter"])
	if err != nil {
		return InsightQuery{}, err
	}
	start, end, err := DateRange(stringParam(params, "duration"), stringParam(params, "from"), stringParam(params, "to"), now, opts.Location)
	if err != nil {
		return InsightQuery{}, err
	}

	q := InsightQuery{StartDate: start, EndDate: end}
	q.Env = strings.Join(FilterValues(filters, FilterEnvironment), ",")
	if opts.CurrentEnvironment {
		q.Env = ""
		q.CurrentEnvironment = true
	}
	q.Workflow = strings.Join(compact(FilterValues(filters, FilterWorkflow)), ",")
	q.Status = strings.Join(FilterValues(filters, FilterStatus), ",")
	q.Selector = strings.Join(compact(FilterValues(filters, FilterLabels)), ",")
	// Tag values may contain commas, so tags travel as a JSON array.
	if tags := compact(FilterValues(filters, FilterTags)); len(tags) > 0 {
		data, _ := json.Marshal(tags)
		q.TagFilter = string(data)
	}

	switch kind {
	case KindPassFail:
		q.Endpoint = EndpointStats
		// The stats endpoint has no status filter; the dashboard sends one and
		// the backend ignores it.
		q.Status = ""
	case KindExecutions:
		q.Endpoint = EndpointExecutions
		q.GroupBy = stringParam(params, "groupBy")
		if q.GroupBy == "" {
			q.GroupBy = "status"
		}
	case KindWorkflows:
		q.Endpoint = EndpointWorkflows
	case KindTimeSeries:
		q.Endpoint = EndpointSeries
		q.Measure = stringParam(params, "measure")
		if q.Measure == "" {
			// The dashboard does not query a time-series report until a
			// measure is chosen.
			return InsightQuery{}, fmt.Errorf("time-series report has no measure; set params.measure")
		}
		q.Aggregate = stringParam(params, "aggregate")
		if q.Aggregate == "" {
			q.Aggregate = "sum"
		}
		q.Segment = stringParam(params, "segment")
		if identity := identityFilters(filters); len(identity) > 0 {
			data, _ := json.Marshal(identity)
			q.IdentityFilters = string(data)
		}
	}
	return q, nil
}

// DateRange resolves a report's duration/from/to into the queried range, as
// the dashboard's getDateParams does:
//   - from and to: used as given;
//   - only from: [from, from + duration];
//   - only to: [to - duration, to];
//   - neither: the duration ending at the start of tomorrow.
//
// A missing or "custom" duration means a week. "Tomorrow" is taken in loc,
// as the dashboard takes it in the browser's time zone; nil means UTC. The
// duration is then subtracted as a fixed number of minutes, as the dashboard
// does, so a range that crosses a daylight-saving change starts an hour off
// local midnight there too.
func DateRange(duration, from, to string, now time.Time, loc *time.Location) (time.Time, time.Time, error) {
	var fromT, toT time.Time
	var err error
	if from != "" {
		if fromT, err = time.Parse(time.RFC3339, from); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("param from must be an RFC3339 timestamp: %w", err)
		}
	}
	if to != "" {
		if toT, err = time.Parse(time.RFC3339, to); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("param to must be an RFC3339 timestamp: %w", err)
		}
	}
	if from != "" && to != "" {
		return fromT, toT, nil
	}

	minutes, ok := durationMinutes[duration]
	if !ok {
		minutes = durationMinutes["week"]
	}
	d := time.Duration(minutes) * time.Minute
	switch {
	case from != "":
		return fromT, fromT.Add(d), nil
	case to != "":
		return toT.Add(-d), toT, nil
	}
	if loc == nil {
		loc = time.UTC
	}
	y, m, day := now.In(loc).Date()
	eod := time.Date(y, m, day+1, 0, 0, 0, 0, loc)
	return eod.Add(-d), eod, nil
}

// identityFilters collects the non-base filter keys of a time-series report,
// a port of getTimeSeriesIdentityFilters.
func identityFilters(filters []Filter) map[string][]string {
	base := map[string]bool{FilterEnvironment: true, FilterWorkflow: true, FilterStatus: true, FilterLabels: true, FilterTags: true}
	result := map[string][]string{}
	for _, f := range filters {
		key := f.FilterConfigurationKey
		if base[key] {
			continue
		}
		if _, done := result[key]; done {
			continue
		}
		if values := compact(FilterValues(filters, key)); len(values) > 0 {
			result[key] = values
		}
	}
	return result
}

func stringParam(params map[string]any, key string) string {
	s, _ := params[key].(string)
	return s
}
