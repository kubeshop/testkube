package v1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	validSpec := func() WorkflowTriggerSpec {
		return WorkflowTriggerSpec{
			Watch: &WorkflowTriggerWatch{
				Resource: WorkflowTriggerResource{Kind: "Deployment", Name: "app"},
			},
			When: WorkflowTriggerWhen{Event: "modified"},
			Run: WorkflowTriggerRun{
				Workflow: WorkflowTriggerWorkflowSelector{Name: "smoke-test"},
			},
		}
	}

	tests := map[string]struct {
		modify     func(*WorkflowTriggerSpec)
		wantErrs   int
		wantSubstr string
	}{
		"valid spec": {
			modify:   func(s *WorkflowTriggerSpec) {},
			wantErrs: 0,
		},

		// watch validation
		"event without watch": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Watch = nil
			},
			wantErrs:   1,
			wantSubstr: "watch is required",
		},
		"watch without kind": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Watch.Resource.Kind = ""
			},
			wantErrs:   1,
			wantSubstr: "watch.resource.kind is required",
		},

		// when validation
		"missing event": {
			modify: func(s *WorkflowTriggerSpec) {
				s.When.Event = ""
			},
			wantErrs:   1,
			wantSubstr: "when.event is required",
		},

		// run validation
		"workflow without selector": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Run.Workflow = WorkflowTriggerWorkflowSelector{}
			},
			wantErrs:   1,
			wantSubstr: "run.workflow must specify",
		},

		// match: value required for comparison operators
		"equals without value": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorEquals},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires a value",
		},
		"not_equals without value": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorNotEquals},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires a value",
		},
		"changed_to without value": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChangedTo},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires a value",
		},
		"changed_from without value": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChangedFrom},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires a value",
		},

		// match: exists/changed don't need value (valid)
		"exists without value is valid": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorExists},
				}
			},
			wantErrs: 0,
		},
		"changed without value is valid": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChanged},
				}
			},
			wantErrs: 0,
		},

		// match: change operators require modified event
		"changed with created event": {
			modify: func(s *WorkflowTriggerSpec) {
				s.When.Event = "created"
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChanged},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires when.event to be \"modified\"",
		},
		"changed_to with deleted event": {
			modify: func(s *WorkflowTriggerSpec) {
				s.When.Event = "deleted"
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChangedTo, Value: "5"},
				}
			},
			wantErrs:   1,
			wantSubstr: "requires when.event to be \"modified\"",
		},
		"changed with modified event is valid": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: FieldOperatorChanged},
				}
			},
			wantErrs: 0,
		},

		// match: empty path
		"empty path": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: "", Operator: FieldOperatorExists},
				}
			},
			wantErrs:   1,
			wantSubstr: "path is required",
		},

		// match: unknown operator
		"unknown operator": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Match = []WorkflowTriggerFieldCondition{
					{Path: ".spec.replicas", Operator: "bogus"},
				}
			},
			wantErrs:   1,
			wantSubstr: "unknown operator",
		},

		// multiple errors
		"multiple errors at once": {
			modify: func(s *WorkflowTriggerSpec) {
				s.Watch = nil
				s.Run.Workflow = WorkflowTriggerWorkflowSelector{}
			},
			wantErrs: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validSpec()
			tc.modify(&spec)
			errs := spec.Validate()
			assert.Len(t, errs, tc.wantErrs, "errors: %v", errs)
			if tc.wantSubstr != "" && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), tc.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", tc.wantSubstr, errs)
				}
			}
		})
	}
}
