package formatters

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kubeshop/testkube/pkg/mcp/boards"
)

// maxRenderedWorkflows bounds the workflows report, which lists every workflow
// in range and can be large.
const maxRenderedWorkflows = 25

// --- list_boards -------------------------------------------------------------

type boardSummaryInput struct {
	ID             string `json:"id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	UpdatedAt      string `json:"updatedAt"`
	Creator        string `json:"creator"`
	Shared         bool   `json:"shared"`
	IsUserFavorite bool   `json:"isUserFavorite"`
	IsOrgFavorite  bool   `json:"isOrgFavorite"`
	AnalysisCount  int    `json:"analysisCount"`
}

type formattedBoardSummary struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Shared      bool   `json:"shared"`
	Reports     int    `json:"reports"`
	Pinned      bool   `json:"pinned,omitempty"`
	OrgPinned   bool   `json:"orgPinned,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// FormatBoardList compacts a board list. The Control Plane reports the total
// in a response header, which the tool does not see, so hasMore is inferred
// from a full page.
func FormatBoardList(raw string, page, pageSize int) (string, error) {
	input, isEmpty, err := ParseJSON[[]boardSummaryInput](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(input) == 0 {
		if page > 0 {
			return "No more boards.", nil
		}
		return "No boards found.", nil
	}

	result := make([]formattedBoardSummary, 0, len(input))
	for _, b := range input {
		result = append(result, formattedBoardSummary{
			ID:          b.ID,
			Slug:        b.Slug,
			Name:        b.Name,
			Description: b.Description,
			Shared:      b.Shared,
			Reports:     b.AnalysisCount,
			Pinned:      b.IsUserFavorite,
			OrgPinned:   b.IsOrgFavorite,
			UpdatedAt:   b.UpdatedAt,
		})
	}

	return FormatJSON(struct {
		Boards  []formattedBoardSummary `json:"boards"`
		Page    int                     `json:"page"`
		HasMore bool                    `json:"hasMore"`
	}{Boards: result, Page: page, HasMore: pageSize > 0 && len(input) >= pageSize})
}

// --- get_board ---------------------------------------------------------------

// FormattedBoard is the compact view of a board the board tools return.
type FormattedBoard struct {
	ID          string          `json:"id"`
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Shared      bool            `json:"shared"`
	Creator     string          `json:"creator,omitempty"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
	Reports     []boards.Report `json:"reports"`
	Layout      [][]string      `json:"layout"`
	Unplaced    []string        `json:"unplaced,omitempty"`
}

// BoardView builds the compact view of a board: reports in the order the
// dashboard shows them, the layout as rows of report IDs, and the reports the
// layout leaves out (which the dashboard does not show).
func BoardView(b *boards.Board) (FormattedBoard, error) {
	ordered, unplaced, err := b.OrderedReports()
	if err != nil {
		return FormattedBoard{}, err
	}
	layout, err := boards.ParseLayout(b.Layout)
	if err != nil {
		return FormattedBoard{}, err
	}
	if ordered == nil {
		ordered = []boards.Report{}
	}
	return FormattedBoard{
		ID:          b.ID,
		Slug:        b.Slug,
		Name:        b.Name,
		Description: b.Description,
		Shared:      b.Shared,
		Creator:     b.Creator,
		UpdatedAt:   b.UpdatedAt,
		Reports:     ordered,
		Layout:      layout.RowIDs(),
		Unplaced:    unplaced,
	}, nil
}

// FormatBoard formats a board response.
func FormatBoard(raw string) (string, error) {
	b, err := boards.ParseBoard(raw)
	if err != nil {
		return "", err
	}
	view, err := BoardView(b)
	if err != nil {
		return "", err
	}
	return FormatJSON(view)
}

// --- render_board ------------------------------------------------------------

type statsSeriesInput struct {
	Total  float64           `json:"total"`
	Values []json.RawMessage `json:"values"`
}

type passFailInput struct {
	RatioStats  statsSeriesInput `json:"ratioStats"`
	TotalStats  statsSeriesInput `json:"totalStats"`
	FailedStats statsSeriesInput `json:"failedStats"`
}

type groupedInput struct {
	Total  float64           `json:"total"`
	Values []json.RawMessage `json:"values"`
}

type executionsInput struct {
	Count    groupedInput `json:"count"`
	Duration groupedInput `json:"duration"`
}

type workflowSummaryInput struct {
	Name                 string `json:"name"`
	TotalExecutionCount  int    `json:"totalExecutionCount"`
	FailedExecutionCount int    `json:"failedExecutionCount"`
	AverageDuration      int64  `json:"averageDuration"`
	P95Duration          int64  `json:"p95Duration"`
	LastRunAt            string `json:"lastRunAt"`
}

type formattedWorkflowSummary struct {
	Name          string  `json:"name"`
	Executions    int     `json:"executions"`
	Failed        int     `json:"failed"`
	PassRate      float64 `json:"passRate"`
	AvgDurationMs int64   `json:"avgDurationMs"`
	P95DurationMs int64   `json:"p95DurationMs"`
	LastRunAt     string  `json:"lastRunAt,omitempty"`
}

// ReportData summarizes the response of the query that renders a report, for
// the value that report shows. measure is the report's params.measure; an
// empty measure means the kind's default.
func ReportData(kind, measure, raw string, maxSamples int) (any, error) {
	if maxSamples <= 0 {
		maxSamples = defaultInsightSeriesSamples
	}
	switch kind {
	case boards.KindPassFail:
		input, _, err := ParseJSON[passFailInput](raw)
		if err != nil {
			return nil, err
		}
		selected := input.RatioStats
		switch measure {
		case "failed-count":
			selected = input.FailedStats
		case "total-count":
			selected = input.TotalStats
		default:
			measure = "ratio"
		}
		points := tuples(selected.Values)
		return map[string]any{
			"measure":          measure,
			"total":            selected.Total,
			"passRatio":        input.RatioStats.Total,
			"totalExecutions":  input.TotalStats.Total,
			"failedExecutions": input.FailedStats.Total,
			"points":           len(points),
			"samples":          downsampleInsightPoints(points, maxSamples),
		}, nil

	case boards.KindExecutions:
		input, _, err := ParseJSON[executionsInput](raw)
		if err != nil {
			return nil, err
		}
		selected := input.Count
		if measure == "duration" {
			selected = input.Duration
		} else {
			measure = "count"
		}
		groups := make([]map[string]any, 0, len(selected.Values))
		for _, t := range tuples(selected.Values) {
			groups = append(groups, map[string]any{"group": t[0], "value": t[1]})
		}
		unit := "executions"
		if measure == "duration" {
			unit = "avg ms"
		}
		return map[string]any{"measure": measure, "unit": unit, "total": selected.Total, "groups": groups}, nil

	case boards.KindWorkflows:
		input, _, err := ParseJSON[[]workflowSummaryInput](raw)
		if err != nil {
			return nil, err
		}
		sort.SliceStable(input, func(i, j int) bool {
			return input[i].TotalExecutionCount > input[j].TotalExecutionCount
		})
		count := len(input)
		if len(input) > maxRenderedWorkflows {
			input = input[:maxRenderedWorkflows]
		}
		workflows := make([]formattedWorkflowSummary, 0, len(input))
		for _, w := range input {
			passRate := 0.0
			if w.TotalExecutionCount > 0 {
				passRate = float64(w.TotalExecutionCount-w.FailedExecutionCount) / float64(w.TotalExecutionCount) * 100
			}
			workflows = append(workflows, formattedWorkflowSummary{
				Name:          w.Name,
				Executions:    w.TotalExecutionCount,
				Failed:        w.FailedExecutionCount,
				PassRate:      passRate,
				AvgDurationMs: w.AverageDuration,
				P95DurationMs: w.P95Duration,
				LastRunAt:     w.LastRunAt,
			})
		}
		return map[string]any{"workflowCount": count, "truncated": count > len(workflows), "workflows": workflows}, nil

	case boards.KindTimeSeries:
		data, _, err := ParseJSON[[]insightSeriesDatum](raw)
		if err != nil {
			return nil, err
		}
		total, segments := summarizeInsightSeries(data, maxSamples)
		if segments == nil {
			segments = []formattedInsightSegment{}
		}
		return map[string]any{"pointCount": total, "series": segments}, nil
	}
	return nil, fmt.Errorf("unknown report kind %q", kind)
}

// tuples decodes the [label, value] pairs the reporting endpoints return.
// Pairs that are not two elements long are skipped.
func tuples(values []json.RawMessage) [][2]any {
	out := make([][2]any, 0, len(values))
	for _, v := range values {
		var pair []any
		if json.Unmarshal(v, &pair) != nil || len(pair) != 2 {
			continue
		}
		out = append(out, [2]any{pair[0], pair[1]})
	}
	return out
}
