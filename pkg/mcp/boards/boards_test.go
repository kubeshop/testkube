package boards

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// translationCase pins the query the dashboard issues for a report. The cases
// live in a JSON fixture so the dashboard's own tests can replay them.
type translationCase struct {
	Name     string            `json:"name"`
	Kind     string            `json:"kind"`
	Now      time.Time         `json:"now"`
	TimeZone string            `json:"timeZone"`
	Params   map[string]any    `json:"params"`
	Endpoint string            `json:"endpoint"`
	Query    map[string]string `json:"query"`
}

func TestBuildQuery_TranslationCases(t *testing.T) {
	data, err := os.ReadFile("testdata/translation_cases.json")
	require.NoError(t, err)
	var cases []translationCase
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			loc, err := ParseTimeZone(tc.TimeZone)
			require.NoError(t, err)
			q, err := BuildQuery(tc.Kind, tc.Params, QueryOptions{Now: tc.Now, Location: loc})
			require.NoError(t, err)
			assert.Equal(t, tc.Endpoint, string(q.Endpoint))
			assert.Equal(t, tc.Query, q.QueryParams())
		})
	}
}

func TestBuildQuery_CurrentEnvironment(t *testing.T) {
	params := map[string]any{
		"filter": []any{map[string]any{"filterConfigurationKey": "environment", "id": "1", "operator": "is", "value": []any{"tkcenv_other"}}},
	}

	q, err := BuildQuery(KindWorkflows, params, QueryOptions{CurrentEnvironment: true})
	require.NoError(t, err)
	assert.True(t, q.CurrentEnvironment)
	assert.Empty(t, q.Env, "the report's own environment filter must not leak into a current-environment query")
	assert.Equal(t, "tkcenv_mine", q.WithEnvironment("tkcenv_mine").QueryParams()["env"])

	q, err = BuildQuery(KindWorkflows, params, QueryOptions{})
	require.NoError(t, err)
	assert.Equal(t, "tkcenv_other", q.WithEnvironment("tkcenv_mine").Env, "WithEnvironment only applies to current-environment queries")
}

func TestBuildQuery_Errors(t *testing.T) {
	_, err := BuildQuery("pie", nil, QueryOptions{})
	assert.ErrorContains(t, err, "unknown report kind")

	_, err = BuildQuery(KindTimeSeries, map[string]any{}, QueryOptions{})
	assert.ErrorContains(t, err, "no measure")

	_, err = BuildQuery(KindWorkflows, map[string]any{"from": "yesterday"}, QueryOptions{})
	assert.ErrorContains(t, err, "RFC3339")
}

func TestNormalizeReport_Defaults(t *testing.T) {
	tests := []struct {
		kind string
		want map[string]any
	}{
		{KindPassFail, map[string]any{"filter": []any{}, "duration": "month", "measure": "ratio"}},
		{KindExecutions, map[string]any{"filter": []any{}, "groupBy": "status", "measure": "count"}},
		{KindWorkflows, map[string]any{"filter": []any{}, "duration": "month"}},
		{KindTimeSeries, map[string]any{"filter": []any{}, "duration": "week", "measure": "execution-count", "aggregate": "sum", "segment": "status", "chartType": "bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got, err := NormalizeReport(tt.kind, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeReport_TimeSeriesMeasureDefaults(t *testing.T) {
	// As the dashboard does when a measure is chosen: its preferred aggregate,
	// segmented by status.
	got, err := NormalizeReport(KindTimeSeries, map[string]any{"measure": "http_req_duration_p95_ms"})
	require.NoError(t, err)
	assert.Equal(t, "max", got["aggregate"])
	assert.Equal(t, "status", got["segment"])

	got, err = NormalizeReport(KindTimeSeries, map[string]any{"measure": "cpu-millicores-min", "aggregate": "max"})
	require.NoError(t, err)
	assert.Equal(t, "max", got["aggregate"], "an offered aggregate the caller chose is kept")

	got, err = NormalizeReport(KindTimeSeries, map[string]any{"segment": ""})
	require.NoError(t, err)
	assert.NotContains(t, got, "segment", "an explicit empty segment means no segment")

	_, err = NormalizeReport(KindTimeSeries, map[string]any{"measure": DefaultTimeSeriesMeasure, "aggregate": "avg"})
	assert.ErrorContains(t, err, `param aggregate "avg" is not offered for measure "execution-count" (allowed: sum)`)
}

func TestAggregateOptions(t *testing.T) {
	tests := []struct {
		measure string
		want    []string
	}{
		{"execution-count", []string{"sum"}},
		{"case-count", []string{"sum"}},
		{"execution-duration", []string{"sum", "avg", "min", "max"}},
		{"cpu-millicores-max", []string{"max", "min"}},
		{"memory-used-min", []string{"min", "max"}},
		{"network-in-avg", []string{"avg"}},
		{"disk-write-total", []string{"sum"}},
		{"http_req_duration_min", []string{"min", "max"}},
		{"iteration_duration_max_ms", []string{"max", "min"}},
		{"test_count_failed", []string{"sum"}},
		{"http_reqs", []string{"sum"}},
		{"http_req_failed_rate", []string{"max", "avg", "min", "sum"}},
		{"http_req_duration_p95_ms", []string{"max", "avg", "min", "sum"}},
		{"Checkout_Latency_P99", []string{"max", "avg", "min", "sum"}},
		{"cart_value", []string{"sum", "avg", "min", "max"}},
	}
	for _, tt := range tests {
		t.Run(tt.measure, func(t *testing.T) {
			assert.Equal(t, tt.want, AggregateOptions(tt.measure))
			assert.Equal(t, tt.want[0], PreferredAggregate(tt.measure))
		})
	}
}

func TestEditParams(t *testing.T) {
	custom := map[string]any{"duration": "custom", "from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z", "measure": "ratio"}

	t.Run("a preset duration drops the stored range", func(t *testing.T) {
		got, err := EditParams(KindPassFail, custom, map[string]any{"duration": "week"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"duration": "week", "measure": "ratio"}, got)

		// Kept, the range would still decide the window.
		now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
		q, err := BuildQuery(KindPassFail, got, QueryOptions{Now: now})
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), q.EndDate)
	})

	t.Run("a preset duration keeps a from or to set with it", func(t *testing.T) {
		got, err := EditParams(KindPassFail, custom, map[string]any{"duration": "day", "from": "2026-03-01T00:00:00Z"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"duration": "day", "from": "2026-03-01T00:00:00Z", "measure": "ratio"}, got)
	})

	t.Run("a custom duration keeps the stored range", func(t *testing.T) {
		got, err := EditParams(KindPassFail, custom, map[string]any{"duration": "custom"})
		require.NoError(t, err)
		assert.Equal(t, custom, got)
	})

	t.Run("a range without a duration makes it custom", func(t *testing.T) {
		got, err := EditParams(KindWorkflows, map[string]any{"duration": "month"}, map[string]any{"from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z"})
		require.NoError(t, err)
		assert.Equal(t, "custom", got["duration"])

		got, err = EditParams(KindWorkflows, map[string]any{"duration": "month"}, map[string]any{"from": "2026-01-01T00:00:00Z"})
		require.NoError(t, err)
		assert.Equal(t, "month", got["duration"], "only from spans the duration forward, so it stays")
	})

	stored := map[string]any{"measure": "execution-duration", "aggregate": "avg", "segment": "workflow", "chartType": "line"}

	t.Run("a new measure resets the aggregate and segment", func(t *testing.T) {
		got, err := EditParams(KindTimeSeries, stored, map[string]any{"measure": "http_req_duration_p95_ms"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"measure": "http_req_duration_p95_ms", "aggregate": "max", "segment": "status", "chartType": "line"}, got)
	})

	t.Run("a new measure keeps an aggregate and segment set with it", func(t *testing.T) {
		got, err := EditParams(KindTimeSeries, stored, map[string]any{"measure": "cpu-millicores-max", "aggregate": "min", "segment": ""})
		require.NoError(t, err)
		assert.Equal(t, "min", got["aggregate"])
		assert.Equal(t, "", got["segment"])
	})

	t.Run("the same measure changes nothing else", func(t *testing.T) {
		got, err := EditParams(KindTimeSeries, stored, map[string]any{"measure": "execution-duration", "chartType": "bar"})
		require.NoError(t, err)
		assert.Equal(t, "avg", got["aggregate"])
		assert.Equal(t, "workflow", got["segment"])
	})

	t.Run("a measure outside time-series resets nothing", func(t *testing.T) {
		got, err := EditParams(KindExecutions, map[string]any{"measure": "count", "groupBy": "workflow"}, map[string]any{"measure": "duration"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"measure": "duration", "groupBy": "workflow"}, got)
	})

	t.Run("an aggregate the measure does not offer is refused", func(t *testing.T) {
		_, err := EditParams(KindTimeSeries, stored, map[string]any{"measure": DefaultTimeSeriesMeasure, "aggregate": "max"})
		assert.ErrorContains(t, err, `param aggregate "max" is not offered for measure "execution-count"`)

		_, err = EditParams(KindTimeSeries, stored, map[string]any{"aggregate": "last"})
		assert.ErrorContains(t, err, "allowed: sum, avg, min, max")
	})

	t.Run("a stored aggregate the measure does not offer stays editable", func(t *testing.T) {
		odd := map[string]any{"measure": DefaultTimeSeriesMeasure, "aggregate": "last"}
		got, err := EditParams(KindTimeSeries, odd, map[string]any{"chartType": "line"})
		require.NoError(t, err)
		assert.Equal(t, "last", got["aggregate"])
	})
}

func TestNormalizeReport_Validation(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		params map[string]any
		want   string
	}{
		{"unknown kind", "pie", nil, "unknown report kind"},
		{"bad duration", KindWorkflows, map[string]any{"duration": "year"}, "param duration must be one of"},
		{"bad pass-fail measure", KindPassFail, map[string]any{"measure": "count"}, "param measure must be one of ratio"},
		{"bad executions measure", KindExecutions, map[string]any{"measure": "ratio"}, "param measure must be one of count"},
		{"empty groupBy", KindExecutions, map[string]any{"groupBy": " "}, "groupBy must be a non-empty string"},
		{"bad aggregate", KindTimeSeries, map[string]any{"aggregate": "median"}, "param aggregate must be one of"},
		{"bad chart type", KindTimeSeries, map[string]any{"chartType": "pie"}, "param chartType must be one of"},
		{"bad overlay", KindTimeSeries, map[string]any{"overlaySuccessRate": "yes"}, "overlaySuccessRate must be a boolean"},
		{"bad from", KindWorkflows, map[string]any{"from": "2026-01-01"}, "param from must be an RFC3339"},
		{"filter not a list", KindWorkflows, map[string]any{"filter": "workflow=a"}, "param filter must be an array"},
		{"filter without key", KindWorkflows, map[string]any{"filter": []any{map[string]any{"value": "a"}}}, "missing filterConfigurationKey"},
		{"filter with bad value", KindWorkflows, map[string]any{"filter": []any{map[string]any{"filterConfigurationKey": "workflow", "value": 3}}}, "invalid value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeReport(tt.kind, tt.params)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestNormalizeReport_FillsFilterIDsAndOperators(t *testing.T) {
	got, err := NormalizeReport(KindWorkflows, map[string]any{
		"filter": []any{map[string]any{"filterConfigurationKey": "workflow", "value": []any{"a"}}},
	})
	require.NoError(t, err)
	filters, err := ParseFilters(got["filter"])
	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.NotEmpty(t, filters[0].ID)
	assert.Equal(t, "is", filters[0].Operator)
}

func TestNormalizeReport_PreservesUnknownKeys(t *testing.T) {
	got, err := NormalizeReport(KindWorkflows, map[string]any{"futureKey": 1})
	require.NoError(t, err)
	assert.Equal(t, 1, got["futureKey"], "keys written by a newer dashboard must survive an MCP edit")
}

func TestCheckParamKeys(t *testing.T) {
	assert.NoError(t, CheckParamKeys(KindExecutions, map[string]any{"groupBy": "workflow", "measure": "count", "duration": "day"}))
	assert.ErrorContains(t, CheckParamKeys(KindWorkflows, map[string]any{"measure": "count", "grupBy": "x"}), "unknown params for a workflows report: grupBy, measure")
}

func TestApplyFilters(t *testing.T) {
	params := map[string]any{
		"filter": []any{
			map[string]any{"filterConfigurationKey": "workflow", "id": "w", "operator": "is", "value": []any{"old"}},
			map[string]any{"filterConfigurationKey": "status", "id": "s", "operator": "is", "value": []any{"failed"}},
			map[string]any{"filterConfigurationKey": "labels-v2", "id": "l", "operator": "is", "value": []any{"a=b"}},
		},
	}
	require.NoError(t, ApplyFilters(params, map[string][]string{
		"workflow": {"new", " "},
		"status":   {},
		"labels":   {"team=core"},
	}))

	filters, err := ParseFilters(params["filter"])
	require.NoError(t, err)
	assert.Equal(t, []string{"new"}, FilterValues(filters, FilterWorkflow))
	assert.Empty(t, FilterValues(filters, FilterStatus), "an empty list removes the key's filters")
	assert.Equal(t, []string{"team=core"}, FilterValues(filters, FilterLabels), "labels is an alias of labels-v2 and replaces it")
}

func TestLayout(t *testing.T) {
	raw := json.RawMessage(`{"version":1,"rows":[{"id":"r1","cells":[{"id":"a"},{"id":"b"}]},{"id":"r2","cells":[{"id":"c"}]}]}`)
	layout, err := ParseLayout(raw)
	require.NoError(t, err)

	t.Run("without drops the cell and empty rows", func(t *testing.T) {
		assert.Equal(t, [][]string{{"a", "b"}}, layout.Without("c").RowIDs())
		assert.Equal(t, [][]string{{"b"}, {"c"}}, layout.Without("a").RowIDs())
	})

	t.Run("order puts unplaced reports last", func(t *testing.T) {
		placed, unplaced := layout.Order([]string{"c", "d", "b", "a"})
		assert.Equal(t, []string{"a", "b", "c"}, placed)
		assert.Equal(t, []string{"d"}, unplaced)
	})

	t.Run("validate", func(t *testing.T) {
		ok := Layout{Version: 1, Rows: []Row{{Cells: []Cell{{ID: "b"}, {ID: "a"}}}}}
		require.NoError(t, ok.Validate([]string{"a", "b"}))
		assert.NotEmpty(t, ok.Rows[0].ID, "missing row IDs are generated")

		assert.ErrorContains(t, (&Layout{Version: 2}).Validate(nil), "version must be 1")
		assert.ErrorContains(t, (&Layout{Version: 1, Rows: []Row{{Cells: []Cell{{ID: "x"}}}}}).Validate([]string{"a"}), "unknown report")
		assert.ErrorContains(t, (&Layout{Version: 1, Rows: []Row{{Cells: []Cell{{ID: "a"}, {ID: "a"}}}}}).Validate([]string{"a"}), "more than once")
		assert.ErrorContains(t, (&Layout{Version: 1, Rows: []Row{{Cells: []Cell{{ID: "a"}}}}}).Validate([]string{"a", "b"}), "leaves out reports b")
		assert.ErrorContains(t, (&Layout{Version: 1, Rows: []Row{{}}}).Validate(nil), "has no cells")
	})

	t.Run("an empty layout parses as version 1", func(t *testing.T) {
		for _, raw := range []string{"", "null", "{}"} {
			l, err := ParseLayout(json.RawMessage(raw))
			require.NoError(t, err)
			assert.Equal(t, LayoutVersion, l.Version)
			assert.Empty(t, l.Rows)
		}
	})
}

func TestBoardOrderedReports(t *testing.T) {
	b, err := ParseBoard(`{"id":"b1","name":"B","layout":{"version":1,"rows":[{"id":"r","cells":[{"id":"y"}]}]},
		"content":{"reports":[{"id":"x","kind":"workflows"},{"id":"y","kind":"pass-fail"}]}}`)
	require.NoError(t, err)

	ordered, unplaced, err := b.OrderedReports()
	require.NoError(t, err)
	require.Len(t, ordered, 2)
	assert.Equal(t, "y", ordered[0].ID)
	assert.Equal(t, "x", ordered[1].ID)
	assert.Equal(t, []string{"x"}, unplaced)
}

func TestParseTimeZone(t *testing.T) {
	loc, err := ParseTimeZone("")
	require.NoError(t, err)
	assert.Equal(t, time.UTC, loc)

	loc, err = ParseTimeZone(" Asia/Kolkata ")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Kolkata", loc.String())

	_, err = ParseTimeZone("Mars/Olympus")
	assert.ErrorContains(t, err, "IANA time zone")
}

func TestValidateReport_AddsNoDefaults(t *testing.T) {
	// A stored report the dashboard renders from its missing params: an edit
	// must not fill in creation defaults that render differently.
	tests := []struct {
		kind   string
		params map[string]any
		absent []string
	}{
		{KindPassFail, map[string]any{"measure": "failed-count"}, []string{"duration"}},
		{KindWorkflows, map[string]any{}, []string{"duration"}},
		{KindTimeSeries, map[string]any{"measure": DefaultTimeSeriesMeasure}, []string{"segment", "duration", "chartType", "aggregate"}},
		{KindExecutions, map[string]any{}, []string{"groupBy", "measure"}},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got, err := ValidateReport(tt.kind, tt.params)
			require.NoError(t, err)
			for _, key := range tt.absent {
				assert.NotContains(t, got, key)
			}
		})
	}

	t.Run("the rendered window is the one the report had", func(t *testing.T) {
		now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
		stored := map[string]any{"measure": "ratio"}
		before, err := BuildQuery(KindPassFail, stored, QueryOptions{Now: now})
		require.NoError(t, err)
		edited, err := ValidateReport(KindPassFail, MergeParams(stored, map[string]any{"measure": "failed-count"}))
		require.NoError(t, err)
		after, err := BuildQuery(KindPassFail, edited, QueryOptions{Now: now})
		require.NoError(t, err)
		assert.Equal(t, before.StartDate, after.StartDate, "an unrelated edit must not change the date window")
	})

	t.Run("it still refuses bad values", func(t *testing.T) {
		_, err := ValidateReport(KindPassFail, map[string]any{"duration": "year"})
		assert.ErrorContains(t, err, "param duration must be one of")
		_, err = ValidateReport(KindTimeSeries, map[string]any{"segment": 3})
		assert.ErrorContains(t, err, "param segment must be a string")
	})
}

func TestParseFilters_LabelShapes(t *testing.T) {
	label := func(operator string, value any) []any {
		return []any{map[string]any{"filterConfigurationKey": FilterLabels, "id": "l", "operator": operator, "value": value}}
	}
	valid := [][]any{
		label("is", []any{"team=core"}),
		label("where", map[string]any{"labelKey": "tier", "labelOperator": "contains", "labelValue": "gold"}),
		label("where", map[string]any{"labelKey": "owner", "labelOperator": "exists"}),
	}
	for _, f := range valid {
		_, err := ParseFilters(f)
		assert.NoError(t, err)
	}
	invalid := map[string][]any{
		"a string with is":        label("is", "team=core"),
		"an array with where":     label("where", []any{"team=core"}),
		"where without a key":     label("where", map[string]any{"labelOperator": "exists"}),
		"where with a bad op":     label("where", map[string]any{"labelKey": "tier", "labelOperator": "equals"}),
		"an unsupported operator": label("contains", []any{"team=core"}),
	}
	for name, f := range invalid {
		t.Run(name, func(t *testing.T) {
			_, err := ParseFilters(f)
			assert.ErrorContains(t, err, "labels-v2")
		})
	}

	t.Run("a stored report with a bad label filter fails to render rather than dropping it", func(t *testing.T) {
		_, err := BuildQuery(KindWorkflows, map[string]any{"filter": label("is", "team=core")}, QueryOptions{})
		assert.ErrorContains(t, err, "invalid value")
	})
}
