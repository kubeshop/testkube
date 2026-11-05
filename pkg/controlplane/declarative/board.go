package declarative

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	"github.com/kubeshop/testkube/pkg/mcp/boards"
)

// maxReportIDLength bounds a report id. Ids the dashboard generates are far
// shorter; the bound only keeps a runaway value out of the layout.
const maxReportIDLength = 128

// Normalize validates an object and fills in its defaults, in place. Every
// problem is reported at once, with the path of the field it is in.
func Normalize(obj Object) error {
	switch o := obj.(type) {
	case *insightsv1.Board:
		return normalizeBoard(o).ToAggregate()
	}
	return fmt.Errorf("unsupported object type %T", obj)
}

func normalizeBoard(b *insightsv1.Board) field.ErrorList {
	errs := validateSlug(b.Name, field.NewPath("metadata", "name"))
	errs = append(errs, normalizeBoardSpec(&b.Spec, b.Name, field.NewPath("spec"))...)
	return errs
}

// boardSlugPattern is the slugs the Control Plane makes from a board's name:
// lower case letters, digits, '-' and '_', starting and ending with a letter
// or digit. It is wider than a DNS-1123 label, which allows no '_', because a
// slug made before declarative boards existed must export and apply again
// unchanged, keeping the board's identity.
var boardSlugPattern = regexp.MustCompile(`^[a-z0-9]([-_a-z0-9]*[a-z0-9])?$`)

// maxBoardSlugLength bounds a slug. The Control Plane does not bound the
// names it makes slugs from, so this is the generous DNS-1123 subdomain bound
// rather than the 63 of a label.
const maxBoardSlugLength = 253

func validateSlug(name string, path *field.Path) field.ErrorList {
	switch {
	case name == "":
		return field.ErrorList{field.Required(path, "the board's slug")}
	case len(name) > maxBoardSlugLength:
		return field.ErrorList{field.TooLong(path, name, maxBoardSlugLength)}
	case !boardSlugPattern.MatchString(name):
		return field.ErrorList{field.Invalid(path, name, "a board slug must consist of lower case letters, digits, '-' or '_', and start and end with a letter or digit (e.g. 'release-health')")}
	}
	return nil
}

func normalizeBoardSpec(spec *insightsv1.BoardSpec, name string, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if strings.TrimSpace(spec.DisplayName) == "" {
		spec.DisplayName = name
	}
	if spec.Shared == nil {
		spec.Shared = ptr.To(true)
	}

	ids := make([]string, 0, len(spec.Reports))
	seen := map[string]bool{}
	for i := range spec.Reports {
		reportPath := path.Child("reports").Index(i)
		report := &spec.Reports[i]
		if idErrs := validateReportID(report.ID, reportPath.Child("id")); len(idErrs) > 0 {
			errs = append(errs, idErrs...)
		} else if seen[report.ID] {
			errs = append(errs, field.Duplicate(reportPath.Child("id"), report.ID))
		} else {
			seen[report.ID] = true
			ids = append(ids, report.ID)
		}
		errs = append(errs, normalizeReport(report, reportPath)...)
	}

	if spec.Layout == nil {
		return errs
	}
	return append(errs, validateLayout(spec.Layout, ids, seen, path.Child("layout"))...)
}

func validateReportID(id string, path *field.Path) field.ErrorList {
	switch {
	case id == "":
		return field.ErrorList{field.Required(path, "the layout refers to reports by id")}
	case len(id) > maxReportIDLength:
		return field.ErrorList{field.TooLong(path, id, maxReportIDLength)}
	case strings.IndexFunc(id, unicode.IsSpace) >= 0:
		return field.ErrorList{field.Invalid(path, id, "must not contain whitespace")}
	}
	return nil
}

func validateLayout(layout *insightsv1.BoardLayout, ids []string, known map[string]bool, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	placed := map[string]bool{}
	for i, row := range layout.Rows {
		rowPath := path.Child("rows").Index(i)
		if len(row.Cells) == 0 {
			errs = append(errs, field.Required(rowPath.Child("cells"), "drop empty rows"))
		}
		for j, id := range row.Cells {
			cellPath := rowPath.Child("cells").Index(j)
			switch {
			case !known[id]:
				errs = append(errs, field.NotFound(cellPath, id))
			case placed[id]:
				errs = append(errs, field.Duplicate(cellPath, id))
			}
			placed[id] = true
		}
	}
	var missing []string
	for _, id := range ids {
		if !placed[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		errs = append(errs, field.Invalid(path.Child("rows"), missing, "every report must be placed; these are not: "+strings.Join(missing, ", ")))
	}
	return errs
}

// normalizeReport validates a report's params and fills in the defaults the
// dashboard gives a new report, with boards.NormalizeReport. Filters without
// an id get a deterministic one first, which boards would otherwise make up
// at random on every apply.
func normalizeReport(report *insightsv1.BoardReport, path *field.Path) field.ErrorList {
	if !boards.IsKind(report.Kind) {
		return field.ErrorList{field.NotSupported(path.Child("kind"), report.Kind, boards.Kinds)}
	}
	paramsPath := path.Child("params")
	params := map[string]any{}
	if report.Params != nil && len(report.Params.Raw) > 0 && string(report.Params.Raw) != "null" {
		if err := json.Unmarshal(report.Params.Raw, &params); err != nil {
			return field.ErrorList{field.Invalid(paramsPath, string(report.Params.Raw), "must be an object")}
		}
	}
	if err := boards.CheckParamKeys(report.Kind, params); err != nil {
		return field.ErrorList{field.Invalid(paramsPath, keysOf(params), err.Error())}
	}
	assignFilterIDs(params, report.ID)
	renderedSum := keepRenderedAggregate(report.Kind, params)
	normalized, err := boards.NormalizeReport(report.Kind, params)
	if err != nil {
		return field.ErrorList{field.Invalid(paramsPath, "", err.Error())}
	}
	if renderedSum {
		normalized["aggregate"] = renderedAggregate
	}
	keepNoSegment(report.Kind, normalized)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return field.ErrorList{field.InternalError(paramsPath, err)}
	}
	report.Params = &apiextensionsv1.JSON{Raw: raw}
	return nil
}

// keepNoSegment writes back the empty segment of a time-series report that
// boards.NormalizeReport dropped. NormalizeReport reads "" as "no grouping"
// but a missing segment as "group by status", so without this a second
// Normalize - the CLI normalizes before sending and the Control Plane again
// on receipt - would group a chart its author left ungrouped.
func keepNoSegment(kind string, params map[string]any) {
	if kind != boards.KindTimeSeries {
		return
	}
	if _, ok := params["segment"]; !ok {
		params["segment"] = ""
	}
}

// keepRenderedAggregate reports whether a time-series report asks for the
// sum its measure does not offer, and if so takes it out of params for
// boards.NormalizeReport, which would refuse it. The dashboard renders a
// stored report without an aggregate with the sum whatever its measure, and
// an export says so (see renderedDefaults), so the export must apply again.
func keepRenderedAggregate(kind string, params map[string]any) bool {
	if kind != boards.KindTimeSeries || params["aggregate"] != renderedAggregate {
		return false
	}
	measure, _ := params["measure"].(string)
	if measure == "" || slices.Contains(boards.AggregateOptions(measure), renderedAggregate) {
		return false
	}
	delete(params, "aggregate")
	return true
}

func assignFilterIDs(params map[string]any, reportID string) {
	filters, ok := params["filter"].([]any)
	if !ok {
		return
	}
	for i, f := range filters {
		entry, ok := f.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := entry["id"].(string); id == "" {
			entry["id"] = fmt.Sprintf("%s-%d", reportID, i)
		}
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// StoredBoard is a board in the shape the Control Plane stores it, as the
// boards package describes it.
type StoredBoard struct {
	Slug        string
	Name        string
	Description string
	Shared      bool
	Layout      boards.Layout
	Reports     []boards.Report
}

// BoardToStored maps a normalized board to its stored form. A board without
// a layout gets a row per report. Row ids are derived from the row's position
// so that applying the same board twice stores the same layout.
func BoardToStored(b *insightsv1.Board) (StoredBoard, error) {
	out := StoredBoard{
		Slug:        b.Name,
		Name:        b.Spec.DisplayName,
		Description: b.Spec.Description,
		Shared:      b.Spec.Shared == nil || *b.Spec.Shared,
		Layout:      boards.Layout{Version: boards.LayoutVersion, Rows: []boards.Row{}},
		Reports:     make([]boards.Report, 0, len(b.Spec.Reports)),
	}
	for _, r := range b.Spec.Reports {
		params := map[string]any{}
		if r.Params != nil && len(r.Params.Raw) > 0 {
			if err := json.Unmarshal(r.Params.Raw, &params); err != nil {
				return StoredBoard{}, fmt.Errorf("report %q: params must be an object: %w", r.ID, err)
			}
		}
		out.Reports = append(out.Reports, boards.Report{ID: r.ID, Kind: r.Kind, Name: r.Name, Description: r.Description, Params: params})
	}
	for i, row := range layoutRows(b.Spec) {
		cells := make([]boards.Cell, 0, len(row.Cells))
		for _, id := range row.Cells {
			cells = append(cells, boards.Cell{ID: id})
		}
		out.Layout.Rows = append(out.Layout.Rows, boards.Row{ID: fmt.Sprintf("row-%d", i), Cells: cells})
	}
	return out, nil
}

// BoardFromStored maps a stored board to the object an export returns. It
// loses nothing: a report the stored layout does not place (which the
// dashboard does not show) gets a row of its own after the placed ones, so
// the export still applies. A layout that is a row per report, in order, is
// left out, as that is what applying a board without one stores.
func BoardFromStored(b *boards.Board, status commonv1.ControlPlaneResourceStatus) (*insightsv1.Board, error) {
	out := &insightsv1.Board{
		Spec: insightsv1.BoardSpec{
			DisplayName: b.Name,
			Description: b.Description,
			Shared:      ptr.To(b.Shared),
		},
		Status: status,
	}
	out.SetGroupVersionKind(insightsv1.GroupVersion.WithKind(insightsv1.BoardKind))
	out.Name = b.Slug
	if out.Status.ID == "" {
		out.Status.ID = b.ID
	}
	for _, r := range b.Content.Reports {
		report := insightsv1.BoardReport{ID: r.ID, Kind: r.Kind, Name: r.Name, Description: r.Description}
		params := withRenderedDefaults(r.Kind, r.Params)
		if len(params) > 0 {
			raw, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("report %q: %w", r.ID, err)
			}
			report.Params = &apiextensionsv1.JSON{Raw: raw}
		}
		out.Spec.Reports = append(out.Spec.Reports, report)
	}

	stored, err := boards.ParseLayout(b.Layout)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, id := range b.ReportIDs() {
		known[id] = true
	}
	layout := &insightsv1.BoardLayout{Rows: []insightsv1.BoardLayoutRow{}}
	placed := map[string]bool{}
	for _, row := range stored.Rows {
		var cells []string
		for _, c := range row.Cells {
			if known[c.ID] && !placed[c.ID] {
				cells = append(cells, c.ID)
				placed[c.ID] = true
			}
		}
		if len(cells) > 0 {
			layout.Rows = append(layout.Rows, insightsv1.BoardLayoutRow{Cells: cells})
		}
	}
	for _, id := range b.ReportIDs() {
		if !placed[id] {
			layout.Rows = append(layout.Rows, insightsv1.BoardLayoutRow{Cells: []string{id}})
		}
	}
	if !isDefaultLayout(layout, out.Spec) {
		out.Spec.Layout = layout
	}
	return out, nil
}

// renderedAggregate is the aggregate the dashboard renders a time-series
// report without one with, whatever its measure.
const renderedAggregate = "sum"

// renderedDefaults are what the dashboard shows for a param a stored report
// does not have, where boards.NormalizeReport would give a new report
// something else. An export writes them out, so applying it shows the same:
//   - a pass-fail or workflows report without a duration shows a week, but is
//     created with a month;
//   - a time-series report without an aggregate shows the sum, but is created
//     with its measure's preferred aggregate;
//   - a time-series report without a segment is ungrouped, but is created
//     grouped by status.
var renderedDefaults = map[string]map[string]any{
	boards.KindPassFail:   {"duration": "week"},
	boards.KindWorkflows:  {"duration": "week"},
	boards.KindTimeSeries: {"aggregate": renderedAggregate, "segment": ""},
}

// withRenderedDefaults returns the params of a stored report with the
// rendered defaults it lacks, leaving the stored params unchanged. An empty
// or null value is lacking too: the dashboard renders it as unset, while
// boards.NormalizeReport would replace it with the creation default. (For a
// segment, "" is itself the rendered default, so it is kept.)
func withRenderedDefaults(kind string, params map[string]any) map[string]any {
	var out map[string]any
	for key, value := range renderedDefaults[kind] {
		if current, ok := params[key]; ok && current != nil && current != "" {
			continue
		}
		if out == nil {
			out = maps.Clone(params)
			if out == nil {
				out = map[string]any{}
			}
		}
		out[key] = value
	}
	if out == nil {
		return params
	}
	return out
}

// layoutRows returns the board's layout, or the default one when it has none.
func layoutRows(spec insightsv1.BoardSpec) []insightsv1.BoardLayoutRow {
	if spec.Layout != nil {
		return spec.Layout.Rows
	}
	rows := make([]insightsv1.BoardLayoutRow, 0, len(spec.Reports))
	for _, r := range spec.Reports {
		rows = append(rows, insightsv1.BoardLayoutRow{Cells: []string{r.ID}})
	}
	return rows
}

func isDefaultLayout(layout *insightsv1.BoardLayout, spec insightsv1.BoardSpec) bool {
	if len(layout.Rows) != len(spec.Reports) {
		return false
	}
	for i, row := range layout.Rows {
		if len(row.Cells) != 1 || row.Cells[0] != spec.Reports[i].ID {
			return false
		}
	}
	return true
}
