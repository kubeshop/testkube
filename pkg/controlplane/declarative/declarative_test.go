package declarative

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	"github.com/kubeshop/testkube/pkg/mcp/boards"
)

func decodeFile(t *testing.T, name string) ([]Object, error) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return Decode(bytes.NewReader(data))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

func TestFixtures_Valid(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "valid", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			objects, err := decodeFile(t, filepath.Join("valid", filepath.Base(file)))
			require.NoError(t, err)
			require.NotEmpty(t, objects)
			for _, obj := range objects {
				require.NoError(t, Normalize(obj))
				// Normalizing again changes nothing, which is what lets an
				// export be re-applied without a diff.
				once := mustJSON(t, obj)
				require.NoError(t, Normalize(obj))
				assert.JSONEq(t, once, mustJSON(t, obj))
			}
		})
	}
}

func TestFixtures_Invalid(t *testing.T) {
	cases := map[string][]string{
		"board-bad-slug.yaml":               {"metadata.name", "lower case letters, digits, '-' or '_'"},
		"board-unknown-report-kind.yaml":    {"spec.reports[0].kind", "pie-chart"},
		"board-unknown-param.yaml":          {"spec.reports[0].params", "unknown params for a pass-fail report: durration"},
		"board-bad-param-value.yaml":        {"spec.reports[0].params", "duration must be one of"},
		"board-duplicate-report.yaml":       {"spec.reports[1].id", "Duplicate value"},
		"board-bad-label.yaml":              {"metadata.labels", "team lead"},
		"board-layout-unknown-report.yaml":  {"spec.layout.rows[0].cells[1]", "ghost"},
		"board-layout-unplaced-report.yaml": {"spec.layout.rows", "these are not: b"},
		"board-unknown-field.yaml":          {"document 1", `unknown field "dispayName"`},
		"unknown-kind.yaml":                 {`unknown kind "Dashboard"`},
		"not-a-controlplane-resource.yaml":  {`"testworkflows.testkube.io/v1" is not a Control Plane resource`},
		"ai-agent-literal-secret.yaml":      {"spec.mcpServers[0].credentialReference", "a key itself never goes in a manifest"},
		"ai-agent-bad-link.yaml": {
			"spec.mcpServers[0].credentialReference", "only a shared_key link",
			"spec.mcpServers[1].name", "Duplicate value",
			"spec.access.roles[0]", `"owner"`,
		},
		"ai-agent-no-instructions.yaml": {"metadata.name", "spec.instructions"},
		"ai-agent-bad-skills.yaml": {
			"spec.skills[0].name", "lowercase letters, digits and single hyphens",
			"spec.skills[1].description: Required", "spec.skills[1].content: Required",
			"spec.skills[2].name", "Duplicate value",
		},
		"llm-provider-credential-header.yaml": {
			"spec.extraHeaders: Forbidden: has a header", "spec.models[0].queryParams: Forbidden: has a query parameter", "Environment Credential",
			"spec.baseUrl", "has a query parameter that looks like a credential",
		},
		"llm-provider-bad-model.yaml": {
			"spec.extraHeaders[Host]", "does not forward",
			"spec.models[0].name", "spec.models[2].name", "Duplicate value",
		},
		"mcp-server-builtin-name.yaml":    {"metadata.name", "built-in Testkube MCP server"},
		"mcp-server-missing-url.yaml":     {"spec.url", "spec.command", "only a stdio server"},
		"mcp-server-url-credentials.yaml": {"spec.url", "must not carry credentials"},
		"test-bundle-bad-skills.yaml": {
			"metadata.name", "may not be more than 59 bytes",
			"spec.skills[0].name", "lowercase letters, digits and single hyphens",
			"spec.skills[1].description: Required", "spec.skills[1].content: Required",
			"spec.skills[2].name", "Duplicate value",
			"spec.contexts[0].title: Required", "spec.contexts[1].content: Required",
			"spec.aiAgents[1]", "spec.aiAgents[2]",
		},
		"test-bundle-git-credentials.yaml": {"spec.gitSource.uri", "must not carry credentials"},
	}
	files, err := filepath.Glob(filepath.Join("testdata", "invalid", "*.yaml"))
	require.NoError(t, err)
	require.Len(t, files, len(cases), "every invalid fixture needs a case")
	for name, wants := range cases {
		t.Run(name, func(t *testing.T) {
			objects, err := decodeFile(t, filepath.Join("invalid", name))
			if err == nil {
				for _, obj := range objects {
					if err = Normalize(obj); err != nil {
						break
					}
				}
			}
			require.Error(t, err)
			for _, want := range wants {
				assert.Contains(t, err.Error(), want)
			}
			// The secrets the fixtures hold must not reach the error, which the
			// CLI prints to terminal and CI logs.
			for _, secret := range []string{"ghp_abc123", "hunter2", "sk-live-123"} {
				assert.NotContains(t, err.Error(), secret)
			}
		})
	}
}

func TestNormalize_BoardDefaults(t *testing.T) {
	objects, err := decodeFile(t, "valid/board-minimal.yaml")
	require.NoError(t, err)
	board := objects[0].(*insightsv1.Board)
	require.NoError(t, Normalize(board))

	assert.Equal(t, "release-health", board.Spec.DisplayName)
	assert.Equal(t, ptr.To(true), board.Spec.Shared)
	assert.Nil(t, board.Spec.Layout, "a board without a layout keeps none; storing it gives a row per report")
	// The creation defaults the dashboard gives a pass-fail report.
	assert.JSONEq(t, `{"duration":"month","measure":"ratio","filter":[]}`, string(board.Spec.Reports[0].Params.Raw))
}

func TestNormalize_FilterIDsAreDeterministic(t *testing.T) {
	normalize := func() string {
		objects, err := decodeFile(t, "valid/board-full.yaml")
		require.NoError(t, err)
		require.NoError(t, Normalize(objects[0]))
		return string(objects[0].(*insightsv1.Board).Spec.Reports[0].Params.Raw)
	}
	first := normalize()
	assert.Equal(t, first, normalize())
	assert.Contains(t, first, `"id":"pass-rate-0"`)
}

func TestDecode_MultiDocumentAndList(t *testing.T) {
	objects, err := decodeFile(t, "valid/multi-document.yaml")
	require.NoError(t, err)
	require.Len(t, objects, 2)
	assert.Equal(t, "first", objects[0].GetName())
	assert.Equal(t, "second", objects[1].GetName())

	list, err := Encode(objects, "json")
	require.NoError(t, err)
	assert.Contains(t, string(list), `"kind": "List"`)
	fromList, err := Decode(bytes.NewReader(list))
	require.NoError(t, err)
	assert.Equal(t, mustJSON(t, objects), mustJSON(t, fromList))
}

// The lenient conversion would keep only the second spec, and applying that
// would remove the reports of the first.
func TestDecode_RejectsDuplicateKeys(t *testing.T) {
	_, err := Decode(strings.NewReader(`apiVersion: insights.testkube.io/v1alpha1
kind: Board
metadata:
  name: b
spec:
  reports:
    - id: kept
      kind: pass-fail
spec: {}
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"spec"`)
}

func TestDecode_StrictListEnvelope(t *testing.T) {
	item := `{"apiVersion":"insights.testkube.io/v1alpha1","kind":"Board","metadata":{"name":"b"}}`
	for _, tt := range []struct{ name, list, wantErr string }{
		{"a misspelled items", `{"apiVersion":"v1","kind":"List","itmes":[` + item + `]}`, `unknown field "itmes"`},
		{"an unknown list metadata field", `{"apiVersion":"v1","kind":"List","metadata":{"resourceVersion":"1","colour":"red"},"items":[]}`, `unknown field "colour"`},
		{"no items", `{"apiVersion":"v1","kind":"List"}`, "items is required"},
		{"items that are not a list", `{"apiVersion":"v1","kind":"List","items":{}}`, "items must be a list"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tt.list))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}

	objects, err := Decode(strings.NewReader(`{"apiVersion":"v1","kind":"List","metadata":{"resourceVersion":""},"items":[` + item + `]}`))
	require.NoError(t, err, "the standard list metadata is allowed")
	assert.Len(t, objects, 1)
}

func TestDecode_MissingTypeMeta(t *testing.T) {
	_, err := Decode(strings.NewReader("metadata:\n  name: x\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion and kind are required")
}

func TestEncode_YAMLRoundTrip(t *testing.T) {
	objects, err := decodeFile(t, "valid/board-full.yaml")
	require.NoError(t, err)
	require.NoError(t, Normalize(objects[0]))

	data, err := Encode(objects, "yaml")
	require.NoError(t, err)
	assert.NotContains(t, string(data), "creationTimestamp")
	assert.NotContains(t, string(data), "status:")

	again, err := Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.JSONEq(t, mustJSON(t, objects[0]), mustJSON(t, again[0]))
}

func TestPrepareForApply(t *testing.T) {
	board := &insightsv1.Board{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "b",
			Namespace:       "testkube",
			UID:             "uid",
			ResourceVersion: "12",
			Labels:          map[string]string{"team": "x"},
			Annotations: map[string]string{
				lastAppliedAnnotation: "{}",
				"note":                "kept",
			},
		},
		Status: commonv1.ControlPlaneResourceStatus{ID: "id", GitOpsOwner: &commonv1.GitOpsOwner{AgentID: "a"}},
	}
	prepared, err := PrepareForApply(board)
	require.NoError(t, err)
	out := prepared.(*insightsv1.Board)
	assert.Equal(t, metav1.ObjectMeta{Name: "b", Labels: map[string]string{"team": "x"}, Annotations: map[string]string{"note": "kept"}}, out.ObjectMeta)
	assert.Equal(t, commonv1.ControlPlaneResourceStatus{}, out.Status)
	assert.Equal(t, insightsv1.GroupVersion.WithKind(insightsv1.BoardKind), out.GroupVersionKind())
	assert.Equal(t, "testkube", board.Namespace, "the input is left alone")
}

func TestBoardStoredRoundTrip(t *testing.T) {
	for _, file := range []string{"valid/board-minimal.yaml", "valid/board-full.yaml"} {
		t.Run(file, func(t *testing.T) {
			objects, err := decodeFile(t, file)
			require.NoError(t, err)
			board := objects[0].(*insightsv1.Board)
			require.NoError(t, Normalize(board))

			stored, err := BoardToStored(board)
			require.NoError(t, err)
			exported, err := BoardFromStored(toBoardsBoard(t, stored), commonv1.ControlPlaneResourceStatus{})
			require.NoError(t, err)
			require.NoError(t, Normalize(exported))

			assert.Equal(t, board.Name, exported.Name)
			assert.Equal(t, board.Labels, exported.Labels)
			assert.JSONEq(t, mustJSON(t, board.Spec), mustJSON(t, exported.Spec))
		})
	}
}

func TestBoardToStored_DefaultLayout(t *testing.T) {
	stored, err := BoardToStored(&insightsv1.Board{
		ObjectMeta: metav1.ObjectMeta{Name: "b"},
		Spec: insightsv1.BoardSpec{Reports: []insightsv1.BoardReport{
			{ID: "a", Kind: boards.KindPassFail},
			{ID: "b", Kind: boards.KindWorkflows},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"a"}, {"b"}}, stored.Layout.RowIDs())
	assert.Equal(t, "row-1", stored.Layout.Rows[1].ID)
	assert.True(t, stored.Shared)
}

func TestBoardFromStored_KeepsUnplacedReports(t *testing.T) {
	stored := &boards.Board{
		ID:     "uuid",
		Slug:   "b",
		Name:   "B",
		Shared: true,
		Layout: json.RawMessage(`{"version":1,"rows":[{"id":"r","cells":[{"id":"x"},{"id":"y"},{"id":"gone"}]}]}`),
	}
	stored.Content.Reports = []boards.Report{{ID: "x", Kind: "pass-fail"}, {ID: "y", Kind: "workflows"}, {ID: "hidden", Kind: "workflows"}}

	exported, err := BoardFromStored(stored, commonv1.ControlPlaneResourceStatus{CreationSource: commonv1.SourceUI})
	require.NoError(t, err)
	require.NotNil(t, exported.Spec.Layout)
	assert.Equal(t, []insightsv1.BoardLayoutRow{{Cells: []string{"x", "y"}}, {Cells: []string{"hidden"}}}, exported.Spec.Layout.Rows)
	assert.Equal(t, "uuid", exported.Status.ID)
	assert.Equal(t, commonv1.SourceUI, exported.Status.CreationSource)
	require.NoError(t, Normalize(exported), "an export applies again")
}

func toBoardsBoard(t *testing.T, stored StoredBoard) *boards.Board {
	t.Helper()
	layout, err := json.Marshal(stored.Layout)
	require.NoError(t, err)
	b := &boards.Board{Slug: stored.Slug, Name: stored.Name, Description: stored.Description, Shared: stored.Shared, Labels: stored.Labels, Layout: layout}
	b.Content.Reports = stored.Reports
	return b
}

func TestLookup(t *testing.T) {
	kind, ok := LookupName("Boards")
	require.True(t, ok)
	assert.Equal(t, insightsv1.BoardKind, kind.Kind)
	assert.Equal(t, ScopeOrganization, kind.Scope)

	_, ok = LookupName("qualitygate")
	assert.False(t, ok)

	got, err := KindOf(&insightsv1.Board{})
	require.NoError(t, err)
	assert.Equal(t, insightsv1.BoardResource, got.Resource)

	resources := map[string]bool{}
	for _, k := range Kinds() {
		assert.False(t, resources[k.Resource], "resource %s is not unique", k.Resource)
		resources[k.Resource] = true
	}
}

// boards.NormalizeReport drops an empty segment and groups a missing one by
// status, so a chart left ungrouped must survive the CLI's Normalize and the
// Control Plane's second one.
func TestNormalize_UngroupedTimeSeriesStaysUngrouped(t *testing.T) {
	board := &insightsv1.Board{
		ObjectMeta: metav1.ObjectMeta{Name: "b"},
		Spec: insightsv1.BoardSpec{Reports: []insightsv1.BoardReport{
			{ID: "none", Kind: boards.KindTimeSeries, Params: &apiextensionsv1.JSON{Raw: []byte(`{"segment":""}`)}},
			{ID: "default", Kind: boards.KindTimeSeries},
		}},
	}
	for range 3 {
		require.NoError(t, Normalize(board))
	}
	assert.Equal(t, "", mustParams(t, board.Spec.Reports[0])["segment"])
	assert.Equal(t, "status", mustParams(t, board.Spec.Reports[1])["segment"])
}

// A chart made ungrouped in the dashboard is stored without a segment, which
// the dashboard shows as no grouping, so the export keeps it ungrouped.
func TestBoardFromStored_UngroupedTimeSeries(t *testing.T) {
	stored := &boards.Board{Slug: "b", Name: "B", Layout: json.RawMessage(`{"version":1,"rows":[]}`)}
	stored.Content.Reports = []boards.Report{
		{ID: "none", Kind: boards.KindTimeSeries, Params: map[string]any{"measure": "execution-count"}},
		{ID: "bare", Kind: boards.KindTimeSeries},
	}
	exported, err := BoardFromStored(stored, commonv1.ControlPlaneResourceStatus{})
	require.NoError(t, err)
	require.NoError(t, Normalize(exported))
	for _, r := range exported.Spec.Reports {
		assert.JSONEq(t, `""`, mustJSON(t, mustParams(t, r)["segment"]), r.ID)
	}
	assert.NotContains(t, stored.Content.Reports[0].Params, "segment", "the stored board is not changed")
}

// A stored report shows the dashboard's rendering defaults for the params it
// lacks, which differ from the creation defaults Normalize gives a new
// report. Stored → export → apply must keep what it shows.
func TestBoardStoredExportApply_KeepsRenderedDefaults(t *testing.T) {
	stored := &boards.Board{Slug: "b", Name: "B", Layout: json.RawMessage(`{"version":1,"rows":[]}`)}
	stored.Content.Reports = []boards.Report{
		{ID: "pass-fail", Kind: boards.KindPassFail},
		{ID: "workflows", Kind: boards.KindWorkflows},
		{ID: "counts", Kind: boards.KindTimeSeries, Params: map[string]any{"measure": "execution-count"}},
		// A measure whose dashboard options do not include sum.
		{ID: "latency", Kind: boards.KindTimeSeries, Params: map[string]any{"measure": "k6-http_req_duration-max"}},
		{ID: "chosen", Kind: boards.KindPassFail, Params: map[string]any{"duration": "quarter"}},
		// Empty and null values render as unset too.
		{ID: "empty-pass-fail", Kind: boards.KindPassFail, Params: map[string]any{"duration": ""}},
		{ID: "null-workflows", Kind: boards.KindWorkflows, Params: map[string]any{"duration": nil}},
		{ID: "empty-aggregate", Kind: boards.KindTimeSeries, Params: map[string]any{"measure": "k6-http_req_duration-max", "aggregate": ""}},
		{ID: "null-aggregate", Kind: boards.KindTimeSeries, Params: map[string]any{"measure": "execution-count", "aggregate": nil, "segment": nil}},
	}
	want := map[string]map[string]any{
		"pass-fail":       {"duration": "week"},
		"workflows":       {"duration": "week"},
		"counts":          {"aggregate": "sum", "segment": ""},
		"latency":         {"aggregate": "sum", "segment": ""},
		"chosen":          {"duration": "quarter"},
		"empty-pass-fail": {"duration": "week"},
		"null-workflows":  {"duration": "week"},
		"empty-aggregate": {"aggregate": "sum"},
		"null-aggregate":  {"aggregate": "sum", "segment": ""},
	}

	exported, err := BoardFromStored(stored, commonv1.ControlPlaneResourceStatus{})
	require.NoError(t, err)
	encoded, err := Encode([]Object{exported}, "yaml")
	require.NoError(t, err)
	applied, err := Decode(bytes.NewReader(encoded))
	require.NoError(t, err)
	// The CLI normalizes before sending, and the Control Plane again.
	for range 2 {
		require.NoError(t, Normalize(applied[0]))
	}
	reapplied, err := BoardToStored(applied[0].(*insightsv1.Board))
	require.NoError(t, err)

	for _, r := range reapplied.Reports {
		for key, value := range want[r.ID] {
			assert.Equal(t, value, r.Params[key], "%s: %s", r.ID, key)
		}
	}
	assert.Empty(t, stored.Content.Reports[0].Params, "the stored board is not changed")
}

// The Control Plane keeps underscores in the slugs it makes from board names,
// so a board made before declarative boards existed may have one; its
// unchanged export must apply again.
func TestBoardStoredExportApply_UnderscoreSlug(t *testing.T) {
	stored := &boards.Board{Slug: "release_health", Name: "release_health", Layout: json.RawMessage(`{"version":1,"rows":[]}`)}
	exported, err := BoardFromStored(stored, commonv1.ControlPlaneResourceStatus{})
	require.NoError(t, err)
	encoded, err := Encode([]Object{exported}, "yaml")
	require.NoError(t, err)
	applied, err := Decode(bytes.NewReader(encoded))
	require.NoError(t, err)
	require.NoError(t, Normalize(applied[0]))
	reapplied, err := BoardToStored(applied[0].(*insightsv1.Board))
	require.NoError(t, err)
	assert.Equal(t, "release_health", reapplied.Slug, "the board keeps its identity")
}

func TestValidateSlug(t *testing.T) {
	for _, slug := range []string{"release-health", "release_health", "a", "2026_q1-report", strings.Repeat("a", maxBoardSlugLength)} {
		assert.Empty(t, validateSlug(slug, field.NewPath("metadata", "name")), slug)
	}
	for _, slug := range []string{"", "Release", "_leading", "trailing-", "has space", "dot.ted", strings.Repeat("a", maxBoardSlugLength+1)} {
		assert.NotEmpty(t, validateSlug(slug, field.NewPath("metadata", "name")), slug)
	}
}

// A new report still gets the creation defaults.
func TestNormalize_NewReportsGetCreationDefaults(t *testing.T) {
	board := &insightsv1.Board{
		ObjectMeta: metav1.ObjectMeta{Name: "b"},
		Spec: insightsv1.BoardSpec{Reports: []insightsv1.BoardReport{
			{ID: "pass-fail", Kind: boards.KindPassFail},
			{ID: "latency", Kind: boards.KindTimeSeries, Params: &apiextensionsv1.JSON{Raw: []byte(`{"measure":"k6-http_req_duration-max"}`)}},
		}},
	}
	require.NoError(t, Normalize(board))
	assert.Equal(t, "month", mustParams(t, board.Spec.Reports[0])["duration"])
	assert.Equal(t, "max", mustParams(t, board.Spec.Reports[1])["aggregate"])
}

func mustParams(t *testing.T, r insightsv1.BoardReport) map[string]any {
	t.Helper()
	var params map[string]any
	require.NoError(t, json.Unmarshal(r.Params.Raw, &params))
	return params
}
