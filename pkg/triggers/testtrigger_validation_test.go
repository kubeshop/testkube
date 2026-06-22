package triggers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// bindToRunner returns ActionParameters that bind the trigger to one runner
// via Target.Match.id. Schema-aware match[] triggers require this binding;
// the fixture lets each test case stay focused on the rule under test.
func bindToRunner(agentID string) *testtriggersv1.TestTriggerActionParameters {
	return &testtriggersv1.TestTriggerActionParameters{
		Target: &commonv1.Target{Match: map[string][]string{"id": {agentID}}},
	}
}

// TestFieldOperatorParity locks the two field-operator enums in lockstep:
//
//   - workflowtriggersv1.WorkflowTriggerFieldOperator (CRD type, hand-edited)
//   - testkube.TestTriggerFieldOperator (DTO, regenerated from OpenAPI by Swagger)
//
// They MUST cover the same string set. If a new operator is added to one and
// forgotten on the other, the matcher rejects it on save (cp-api validator)
// or accepts it on save and silently fails to fire (agent runtime). This
// test catches that drift before either path ships broken.
func TestFieldOperatorParity(t *testing.T) {
	wantValues := []string{"equals", "not_equals", "exists", "not_exists", "changed", "changed_to", "changed_from"}

	crdOperators := []workflowtriggersv1.WorkflowTriggerFieldOperator{
		workflowtriggersv1.FieldOperatorEquals,
		workflowtriggersv1.FieldOperatorNotEquals,
		workflowtriggersv1.FieldOperatorExists,
		workflowtriggersv1.FieldOperatorNotExists,
		workflowtriggersv1.FieldOperatorChanged,
		workflowtriggersv1.FieldOperatorChangedTo,
		workflowtriggersv1.FieldOperatorChangedFrom,
	}
	dtoOperators := []testkube.TestTriggerFieldOperator{
		testkube.TestTriggerFieldOperatorEquals,
		testkube.TestTriggerFieldOperatorNotEquals,
		testkube.TestTriggerFieldOperatorExists,
		testkube.TestTriggerFieldOperatorNotExists,
		testkube.TestTriggerFieldOperatorChanged,
		testkube.TestTriggerFieldOperatorChangedTo,
		testkube.TestTriggerFieldOperatorChangedFrom,
	}

	crdSet := map[string]struct{}{}
	for _, o := range crdOperators {
		crdSet[string(o)] = struct{}{}
	}
	dtoSet := map[string]struct{}{}
	for _, o := range dtoOperators {
		dtoSet[string(o)] = struct{}{}
	}
	wantSet := map[string]struct{}{}
	for _, v := range wantValues {
		wantSet[v] = struct{}{}
	}

	assert.Equal(t, wantSet, crdSet, "CRD operator set drifted from canonical list - update both enums and this test")
	assert.Equal(t, wantSet, dtoSet, "DTO operator set drifted from canonical list - update both enums and this test")
	assert.Equal(t, crdSet, dtoSet, "CRD and DTO operator enums diverged - sync them")
}

// TestMatchPathPattern locks the dot-path syntax cp-api/testkube agree on.
// Real shapes the UI emits today, plus shapes a future array-aware backend
// would emit, plus shapes that should always be rejected.
//
// Lives under pkg/triggers (not api/testtriggers/v1) because the latter has a
// pre-existing Ginkgo suite that fails to load and blocks every test in the
// package - moving here lets the regex still be exercised from a clean suite.
func TestMatchPathPattern(t *testing.T) {
	tests := map[string]struct {
		path  string
		valid bool
	}{
		"simple top-level":           {".kind", true},
		"nested two-deep":            {".status.phase", true},
		"deep nest":                  {".spec.template.spec.containers", true},
		"identifier with hyphen":     {".metadata.app-version", true},
		"identifier with underscore": {".spec.snake_case_field", true},
		"wildcard array":             {".spec.containers[*].image", true},
		"specific index array":       {".spec.containers[0].image", true},
		"chained wildcards":          {".spec.a[*].b[*].c", true},
		"empty":                      {"", false},
		"missing leading dot":        {"spec.replicas", false},
		"trailing dot":               {".spec.", false},
		"double dot":                 {".spec..replicas", false},
		"unbalanced bracket":         {".spec.containers[0.image", false},
		"non-numeric specific index": {".spec.containers[abc].image", false},
		"whitespace inside":          {".spec. replicas", false},
		"emoji segment":              {".spec.💩", false},
		"dollar sign":                {".$ref", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.valid, testtriggersv1.MatchPathPattern.MatchString(tc.path), "path=%q", tc.path)
		})
	}
}

// TestTestTriggerSpecValidate covers the full Validate() output: missing path,
// bad path syntax, missing required value, unknown operator, and the
// change-operators-only-with-modified-event rule.
func TestTestTriggerSpecValidate(t *testing.T) {
	tests := map[string]struct {
		spec       testtriggersv1.TestTriggerSpec
		wantErrMsg string // substring expected in one of the returned errors; "" means no errors
	}{
		"valid scalar equals": {
			spec: testtriggersv1.TestTriggerSpec{
				Event:    testtriggersv1.TestTriggerEventModified,
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorEquals, Value: "Healthy"},
				},
			},
			wantErrMsg: "",
		},
		"valid exists with no value": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "",
		},
		"empty match[] passes without listener binding": {
			spec:       testtriggersv1.TestTriggerSpec{Event: testtriggersv1.TestTriggerEventCreated},
			wantErrMsg: "",
		},
		"match[] without a listener is rejected": {
			spec: testtriggersv1.TestTriggerSpec{
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "listener",
		},
		"match[] with empty listener match.id is rejected": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "listener",
		},
		"match[] with only a runner binding is rejected": {
			spec: testtriggersv1.TestTriggerSpec{
				ActionParameters: bindToRunner("tkcagnt_test"),
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "listener",
		},
		"missing path": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: "", Operator: workflowtriggersv1.FieldOperatorEquals, Value: "x"},
				},
			},
			wantErrMsg: "path is required",
		},
		"invalid path syntax": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: "spec.replicas", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "is not a valid dot-path",
		},
		"bracket wildcard path is rejected": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".spec.containers[*].image", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "array-path matching is not supported yet",
		},
		"bracket index path is rejected": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".spec.containers[0].image", Operator: workflowtriggersv1.FieldOperatorExists},
				},
			},
			wantErrMsg: "array-path matching is not supported yet",
		},
		"equals without value": {
			spec: testtriggersv1.TestTriggerSpec{
				Event:    testtriggersv1.TestTriggerEventModified,
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorEquals, Value: ""},
				},
			},
			wantErrMsg: "requires a value",
		},
		"unknown operator": {
			spec: testtriggersv1.TestTriggerSpec{
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: "matches"},
				},
			},
			wantErrMsg: "unknown operator",
		},
		"changed_to with non-modified event": {
			spec: testtriggersv1.TestTriggerSpec{
				Event:    "created",
				Listener: &commonv1.Target{Match: map[string][]string{"id": {"tkcagnt_test"}}},
				Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
					{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorChangedTo, Value: "Healthy"},
				},
			},
			wantErrMsg: `requires event to be "modified"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			errs := tc.spec.Validate()
			if tc.wantErrMsg == "" {
				assert.Empty(t, errs, "expected no errors, got %v", errs)
				return
			}
			assert.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if assert.Contains(t, e.Error(), tc.wantErrMsg) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected an error containing %q in %v", tc.wantErrMsg, errs)
		})
	}
}
