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

func TestNormalizeReport_TimeSeriesSegment(t *testing.T) {
	got, err := NormalizeReport(KindTimeSeries, map[string]any{"measure": "cpu-millicores-max"})
	require.NoError(t, err)
	assert.NotContains(t, got, "segment", "only execution-count starts segmented by status")

	got, err = NormalizeReport(KindTimeSeries, map[string]any{"segment": ""})
	require.NoError(t, err)
	assert.NotContains(t, got, "segment", "an explicit empty segment means no segment")
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
