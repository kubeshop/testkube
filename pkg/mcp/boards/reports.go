// Package boards holds the transport-free logic behind the Insights board MCP
// tools: the shape of a report's params, the translation of a report into the
// insight query that renders it, and the board layout.
//
// Report params have no backend schema - the Control Plane stores them as an
// opaque JSON object and only the dashboard interprets them. The rules here are
// a port of the dashboard's report types and filter mapping, so a report the
// MCP writes renders and edits in the UI, and render_board shows the numbers
// the UI shows. Both the CLI client and the Control Plane's hosted MCP use this
// package; neither side should reimplement it.
package boards

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// Report kinds, matching the dashboard's report registry.
const (
	KindTimeSeries = "time-series"
	KindExecutions = "executions"
	KindPassFail   = "pass-fail"
	KindWorkflows  = "workflows"
)

// Kinds lists every report kind the dashboard can render.
var Kinds = []string{KindTimeSeries, KindExecutions, KindPassFail, KindWorkflows}

// Filter configuration keys the dashboard treats as base filters. Any other key
// on a time-series report is an identity filter on the granular series.
const (
	FilterEnvironment = "environment"
	FilterWorkflow    = "workflow"
	FilterStatus      = "status"
	FilterLabels      = "labels-v2"
	FilterTags        = "tags"
)

var (
	durations          = []string{"day", "week", "month", "quarter"}
	passFailMeasures   = []string{"ratio", "failed-count", "total-count"}
	executionsMeasures = []string{"count", "duration"}
	// The dashboard offers "last" as well; the backend falls back to sum for it.
	timeSeriesAggregates = []string{"sum", "avg", "min", "max", "count", "last"}
	timeSeriesChartTypes = []string{"bar", "bar-grouped", "bar-normalized", "line-stacked", "line", "area-normalized", "heatmap", "horizon"}
)

// DefaultTimeSeriesMeasure is the measure a new time-series report starts with.
const DefaultTimeSeriesMeasure = "execution-count"

// paramKeys lists the params each kind understands.
var paramKeys = map[string][]string{
	KindPassFail:   {"filter", "duration", "from", "to", "measure"},
	KindExecutions: {"filter", "duration", "from", "to", "groupBy", "measure"},
	KindWorkflows:  {"filter", "duration", "from", "to"},
	KindTimeSeries: {"filter", "duration", "from", "to", "measure", "aggregate", "segment", "chartType", "overlaySuccessRate"},
}

// Report is one report ("analysis") on a board, as the Control Plane returns it.
type Report struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
}

// ReportDraft is the content of a report create or update request.
type ReportDraft struct {
	Kind        string         `json:"kind"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Params      map[string]any `json:"params"`
}

// Filter is one entry of a report's params.filter, in the dashboard's format.
type Filter struct {
	FilterConfigurationKey string          `json:"filterConfigurationKey"`
	ID                     string          `json:"id"`
	Operator               string          `json:"operator"`
	Value                  json.RawMessage `json:"value"`
}

// LabelFilterValue is the value of an advanced ("where") label filter.
type LabelFilterValue struct {
	LabelKey      string `json:"labelKey"`
	LabelOperator string `json:"labelOperator"`
	LabelValue    string `json:"labelValue"`
}

// IsKind reports whether kind is a report kind the dashboard can render.
func IsKind(kind string) bool {
	return slices.Contains(Kinds, kind)
}

// CheckParamKeys rejects params the kind does not understand, so a typo in a
// caller's input is reported instead of being stored and silently ignored.
// It is applied to caller input only: a report written by a newer dashboard may
// carry keys this package does not know, and those are preserved.
func CheckParamKeys(kind string, params map[string]any) error {
	allowed, ok := paramKeys[kind]
	if !ok {
		return unknownKindError(kind)
	}
	var unknown []string
	for key := range params {
		if !slices.Contains(allowed, key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown params for a %s report: %s (allowed: %s)", kind, strings.Join(unknown, ", "), strings.Join(allowed, ", "))
	}
	return nil
}

// FiltersToParams converts the convenience filters object ({"workflow": [...],
// "labels": [...], ...}) into dashboard filters. "labels" is accepted as an
// alias of the dashboard's "labels-v2" key. Keys are processed in sorted order
// so the output is deterministic.
func FiltersToParams(filters map[string][]string) []Filter {
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]Filter, 0, len(keys))
	for _, key := range keys {
		values := compact(filters[key])
		if len(values) == 0 {
			continue
		}
		value, _ := json.Marshal(values)
		result = append(result, Filter{
			FilterConfigurationKey: canonicalFilterKey(key),
			ID:                     newID(),
			Operator:               "is",
			Value:                  value,
		})
	}
	return result
}

// ApplyFilters merges convenience filters into params.filter: every key named
// in filters replaces the existing entries for that key, and an empty list
// removes them. Keys not named are left untouched.
func ApplyFilters(params map[string]any, filters map[string][]string) error {
	if len(filters) == 0 {
		return nil
	}
	existing, err := ParseFilters(params["filter"])
	if err != nil {
		return err
	}
	replaced := map[string]bool{}
	for key := range filters {
		replaced[canonicalFilterKey(key)] = true
	}
	merged := make([]Filter, 0, len(existing)+len(filters))
	for _, f := range existing {
		if !replaced[f.FilterConfigurationKey] {
			merged = append(merged, f)
		}
	}
	merged = append(merged, FiltersToParams(filters)...)
	params["filter"] = filtersToAny(merged)
	return nil
}

// NormalizeReport validates the params of a new report and fills in the
// defaults the dashboard gives a new report of that kind, returning a new
// params map. Unknown keys are preserved (see CheckParamKeys).
//
// It is for creating a report, or replacing its params wholesale. To edit an
// existing report use ValidateReport: some creation defaults differ from how
// the dashboard renders a param that is missing (a pass-fail report without a
// duration renders a week but is created with a month), so applying them to
// a stored report would change what it shows.
func NormalizeReport(kind string, params map[string]any) (map[string]any, error) {
	if !IsKind(kind) {
		return nil, unknownKindError(kind)
	}
	out := copyParams(params)
	switch kind {
	case KindPassFail:
		setDefault(out, "duration", "month")
		setDefault(out, "measure", "ratio")
	case KindExecutions:
		setDefault(out, "groupBy", "status")
		setDefault(out, "measure", "count")
	case KindWorkflows:
		setDefault(out, "duration", "month")
	case KindTimeSeries:
		setDefault(out, "duration", "week")
		setDefault(out, "measure", DefaultTimeSeriesMeasure)
		setDefault(out, "aggregate", "sum")
		// A new report is segmented by status only for the measure whose
		// segments are execution statuses; an explicit "" means "no segment".
		if _, ok := out["segment"]; !ok && out["measure"] == DefaultTimeSeriesMeasure {
			out["segment"] = "status"
		}
		setDefault(out, "chartType", "bar")
	}
	if err := validateReport(kind, out); err != nil {
		return nil, err
	}
	// A new report must name what it groups or measures.
	switch kind {
	case KindExecutions:
		if err := checkNonEmptyString(out, "groupBy"); err != nil {
			return nil, err
		}
	case KindTimeSeries:
		if err := checkNonEmptyString(out, "measure"); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ValidateReport validates the params of an existing report after an edit,
// returning a new params map. It checks every param that is set and gives
// filters their IDs, but adds no defaults, so an edit changes only what the
// caller named. Unknown keys are preserved (see CheckParamKeys).
func ValidateReport(kind string, params map[string]any) (map[string]any, error) {
	if !IsKind(kind) {
		return nil, unknownKindError(kind)
	}
	out := copyParams(params)
	if err := validateReport(kind, out); err != nil {
		return nil, err
	}
	return out, nil
}

// validateReport checks the params that are set, in place: it gives filters
// their IDs and operators, drops empty from/to and an empty segment, and adds
// nothing else.
func validateReport(kind string, out map[string]any) error {
	filters, err := ParseFilters(out["filter"])
	if err != nil {
		return err
	}
	for i := range filters {
		if filters[i].ID == "" {
			filters[i].ID = newID()
		}
		if filters[i].Operator == "" {
			filters[i].Operator = "is"
		}
	}
	out["filter"] = filtersToAny(filters)

	if err := checkEnum(out, "duration", append(slices.Clone(durations), "custom")); err != nil {
		return err
	}
	for _, key := range []string{"from", "to"} {
		if err := checkTime(out, key); err != nil {
			return err
		}
	}

	switch kind {
	case KindPassFail:
		return checkEnum(out, "measure", passFailMeasures)
	case KindExecutions:
		if err := checkEnum(out, "measure", executionsMeasures); err != nil {
			return err
		}
		return checkString(out, "groupBy")
	case KindTimeSeries:
		if err := checkString(out, "measure"); err != nil {
			return err
		}
		if _, ok := out["measure"]; ok {
			if err := checkNonEmptyString(out, "measure"); err != nil {
				return err
			}
		}
		if err := checkEnum(out, "aggregate", timeSeriesAggregates); err != nil {
			return err
		}
		if err := checkString(out, "segment"); err != nil {
			return err
		}
		if out["segment"] == "" {
			delete(out, "segment")
		}
		if err := checkEnum(out, "chartType", timeSeriesChartTypes); err != nil {
			return err
		}
		if v, ok := out["overlaySuccessRate"]; ok {
			if _, isBool := v.(bool); !isBool {
				return fmt.Errorf("param overlaySuccessRate must be a boolean, got %T", v)
			}
		}
	}
	return nil
}

func copyParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params)+4)
	for k, v := range params {
		out[k] = v
	}
	return out
}

// MergeParams returns existing params with patch applied on top: keys in patch
// replace the existing ones and a nil value removes a key.
func MergeParams(existing, patch map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(patch))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range patch {
		if v == nil {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

// ParseFilters decodes params.filter. A missing filter is an empty list.
func ParseFilters(raw any) ([]Filter, error) {
	if raw == nil {
		return []Filter{}, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("param filter is malformed: %w", err)
	}
	var filters []Filter
	if err := json.Unmarshal(data, &filters); err != nil {
		return nil, fmt.Errorf("param filter must be an array of {filterConfigurationKey, operator, value} objects: %w", err)
	}
	for i, f := range filters {
		if f.FilterConfigurationKey == "" {
			return nil, fmt.Errorf("param filter[%d] is missing filterConfigurationKey", i)
		}
		if !validFilterValue(f) {
			return nil, fmt.Errorf("param filter[%d] (%s) has an invalid value: expected a string or an array of strings; a labels-v2 filter takes an array of selectors with operator 'is', or {labelKey, labelOperator: exists|contains, labelValue} with operator 'where'", i, f.FilterConfigurationKey)
		}
	}
	return filters, nil
}

// FilterValues gathers every value of the filters with the given key,
// regardless of operator. It is a port of the dashboard's getFilterValue:
// a string value with the "contains" operator becomes "~value", and an
// advanced label filter becomes "key=~value" (contains) or "key" (exists).
func FilterValues(filters []Filter, key string) []string {
	result := []string{}
	for _, f := range filters {
		if f.FilterConfigurationKey != key {
			continue
		}
		if key == FilterLabels {
			switch f.Operator {
			case "is":
				var values []string
				if json.Unmarshal(f.Value, &values) == nil {
					result = append(result, values...)
				}
			case "where":
				var v LabelFilterValue
				if json.Unmarshal(f.Value, &v) != nil {
					continue
				}
				switch v.LabelOperator {
				case "contains":
					if v.LabelKey != "" && v.LabelValue != "" {
						result = append(result, v.LabelKey+"=~"+v.LabelValue)
					}
				case "exists":
					if v.LabelKey != "" {
						result = append(result, v.LabelKey)
					}
				}
			}
			continue
		}
		var s string
		if json.Unmarshal(f.Value, &s) == nil {
			if s == "" {
				continue
			}
			if f.Operator == "contains" {
				result = append(result, "~"+s)
			} else {
				result = append(result, s)
			}
			continue
		}
		var values []string
		if json.Unmarshal(f.Value, &values) == nil {
			result = append(result, values...)
		}
	}
	return result
}

func validFilterValue(f Filter) bool {
	if f.FilterConfigurationKey == FilterLabels {
		// Only the two shapes the dashboard writes, which FilterValues reads:
		// any other value would pass here and then be silently ignored, and
		// the report would query without its label restriction.
		switch f.Operator {
		case "is", "":
			var values []string
			return json.Unmarshal(f.Value, &values) == nil
		case "where":
			var v LabelFilterValue
			if json.Unmarshal(f.Value, &v) != nil || v.LabelKey == "" {
				return false
			}
			return v.LabelOperator == "exists" || (v.LabelOperator == "contains" && v.LabelValue != "")
		}
		return false
	}
	var s string
	if json.Unmarshal(f.Value, &s) == nil {
		return true
	}
	var values []string
	return json.Unmarshal(f.Value, &values) == nil
}

func filtersToAny(filters []Filter) []any {
	out := make([]any, 0, len(filters))
	for _, f := range filters {
		var value any
		_ = json.Unmarshal(f.Value, &value)
		out = append(out, map[string]any{
			"filterConfigurationKey": f.FilterConfigurationKey,
			"id":                     f.ID,
			"operator":               f.Operator,
			"value":                  value,
		})
	}
	return out
}

func canonicalFilterKey(key string) string {
	if key == "labels" {
		return FilterLabels
	}
	return key
}

func compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func setDefault(params map[string]any, key string, value any) {
	if v, ok := params[key]; !ok || v == nil || v == "" {
		params[key] = value
	}
}

func checkEnum(params map[string]any, key string, allowed []string) error {
	v, ok := params[key]
	if !ok || v == "" {
		return nil
	}
	s, isString := v.(string)
	if !isString || !slices.Contains(allowed, s) {
		return fmt.Errorf("param %s must be one of %s, got %v", key, strings.Join(allowed, ", "), v)
	}
	return nil
}

// checkString requires a param, when set, to be a string.
func checkString(params map[string]any, key string) error {
	if v, ok := params[key]; ok {
		if _, isString := v.(string); !isString {
			return fmt.Errorf("param %s must be a string, got %T", key, v)
		}
	}
	return nil
}

func checkNonEmptyString(params map[string]any, key string) error {
	s, ok := params[key].(string)
	if !ok || strings.TrimSpace(s) == "" {
		return fmt.Errorf("param %s must be a non-empty string", key)
	}
	return nil
}

func checkTime(params map[string]any, key string) error {
	v, ok := params[key]
	if !ok || v == nil || v == "" {
		delete(params, key)
		return nil
	}
	s, isString := v.(string)
	if !isString {
		return fmt.Errorf("param %s must be an RFC3339 timestamp string, got %T", key, v)
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return fmt.Errorf("param %s must be an RFC3339 timestamp (e.g. 2026-01-15T00:00:00Z): %w", key, err)
	}
	return nil
}

func unknownKindError(kind string) error {
	return fmt.Errorf("unknown report kind %q (allowed: %s)", kind, strings.Join(Kinds, ", "))
}

// newID returns a random identifier for a filter entry; the dashboard uses it
// only as a stable key when rendering the filter list.
func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("f%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
