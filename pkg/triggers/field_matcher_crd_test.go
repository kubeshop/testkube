package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// argoRollout returns an Argo Rollouts CR shaped as the agent would observe it
// via the dynamic informer (always map[string]interface{} regardless of group).
// Mirrors the schema we push from clusterdiscovery so the path syntax users
// pick in the UI is exercised against the same shape the matcher will see.
func argoRollout(phase string, replicas, paused interface{}) map[string]interface{} {
	return map[string]interface{}{
		"kind":       "Rollout",
		"apiVersion": "argoproj.io/v1alpha1",
		"metadata": map[string]interface{}{
			"name":      "guestbook",
			"namespace": "default",
			"labels":    map[string]interface{}{"app": "guestbook"},
		},
		"spec": map[string]interface{}{
			"replicas": replicas,
			"paused":   paused,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{"app": "guestbook"},
			},
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": "guestbook", "image": "nginx:1.19"},
						map[string]interface{}{"name": "sidecar", "image": "envoy:1.24"},
					},
				},
			},
		},
		"status": map[string]interface{}{
			"phase": phase,
			"conditions": []interface{}{
				map[string]interface{}{"type": "Available", "status": "True"},
				map[string]interface{}{"type": "Progressing", "status": "False"},
			},
		},
	}
}

// TestMatchOnCRDScalarTypes covers all operators against the scalar field
// types CRDs commonly carry (string, int, float, bool). These are the paths
// the UI's TreeSelect surfaces as leaves, and they must round-trip cleanly
// through the matcher.
func TestMatchOnCRDScalarTypes(t *testing.T) {
	old := argoRollout("Progressing", float64(3), false)
	cur := argoRollout("Healthy", float64(5), true)

	tests := map[string]struct {
		path     string
		op       v1.WorkflowTriggerFieldOperator
		value    string
		obj      any
		oldObj   any
		expected bool
	}{
		// Strings (status enums on CRDs)
		"string equals":           {".status.phase", v1.FieldOperatorEquals, "Healthy", cur, nil, true},
		"string equals fails":     {".status.phase", v1.FieldOperatorEquals, "Degraded", cur, nil, false},
		"string not_equals":       {".status.phase", v1.FieldOperatorNotEquals, "Degraded", cur, nil, true},
		"string changed_to":       {".status.phase", v1.FieldOperatorChangedTo, "Healthy", cur, old, true},
		"string changed_from":     {".status.phase", v1.FieldOperatorChangedFrom, "Progressing", cur, old, true},
		"string changed":          {".status.phase", v1.FieldOperatorChanged, "", cur, old, true},
		"string changed_to fails": {".status.phase", v1.FieldOperatorChangedTo, "Degraded", cur, old, false},

		// Integers (the matcher coerces numeric scalars to their string form)
		"int equals":       {".spec.replicas", v1.FieldOperatorEquals, "5", cur, nil, true},
		"int equals fails": {".spec.replicas", v1.FieldOperatorEquals, "3", cur, nil, false},
		"int changed":      {".spec.replicas", v1.FieldOperatorChanged, "", cur, old, true},
		"int changed_to":   {".spec.replicas", v1.FieldOperatorChangedTo, "5", cur, old, true},

		// Booleans
		"bool equals true":        {".spec.paused", v1.FieldOperatorEquals, "true", cur, nil, true},
		"bool equals false":       {".spec.paused", v1.FieldOperatorEquals, "false", old, nil, true},
		"bool changed_to true":    {".spec.paused", v1.FieldOperatorChangedTo, "true", cur, old, true},
		"bool changed_from false": {".spec.paused", v1.FieldOperatorChangedFrom, "false", cur, old, true},

		// Nested maps with simple-identifier keys
		"label equals":         {".metadata.labels.app", v1.FieldOperatorEquals, "guestbook", cur, nil, true},
		"selector matchLabels": {".spec.selector.matchLabels.app", v1.FieldOperatorEquals, "guestbook", cur, nil, true},

		// exists / not_exists
		"exists on present scalar":      {".status.phase", v1.FieldOperatorExists, "", cur, nil, true},
		"not_exists on missing field":   {".status.health", v1.FieldOperatorNotExists, "", cur, nil, true},
		"exists on missing field fails": {".status.health", v1.FieldOperatorExists, "", cur, nil, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			conds := []v1.WorkflowTriggerFieldCondition{{Path: tc.path, Operator: tc.op, Value: tc.value}}
			assert.Equal(t, tc.expected, matchFieldSelector(conds, tc.obj, tc.oldObj))
		})
	}
}

// TestMatchOnCRDArrayPathsCurrentlyUnsupported documents the runtime
// behavior: the matcher's expression engine (pkg/expressions) doesn't tokenize
// `[*]` or `[N]`, so any bracket path it receives fails to compile and the
// condition can't fire.
//
// As of the array-path-rejection change, the cp-api + CRD validators catch
// these paths at save time (see TestTriggerSpec.Validate +
// validateTestTriggerMatch - both reject with "array-path matching is not
// supported yet"). This test only fires if a stored trigger somehow bypasses
// validation (legacy data, direct DB write, mismatched validator version).
// The assertions stay false so when pkg/expressions gains bracket support
// they flip and the suite forces an update across all three layers.
func TestMatchOnCRDArrayPathsCurrentlyUnsupported(t *testing.T) {
	cur := argoRollout("Healthy", float64(5), false)

	gaps := map[string]struct {
		path string
		op   v1.WorkflowTriggerFieldOperator
	}{
		"specific index":             {".spec.template.spec.containers[0].image", v1.FieldOperatorEquals},
		"wildcard any":               {".spec.template.spec.containers[*].image", v1.FieldOperatorEquals},
		"wildcard on conditions":     {".status.conditions[*].status", v1.FieldOperatorEquals},
		"specific cond index exists": {".status.conditions[0].type", v1.FieldOperatorExists},
	}

	for name, g := range gaps {
		t.Run(name, func(t *testing.T) {
			conds := []v1.WorkflowTriggerFieldCondition{{Path: g.path, Operator: g.op, Value: "x"}}
			// Today: any path containing `[…]` fails to compile, so the
			// matcher returns false regardless of the data underneath.
			assert.False(t, matchFieldSelector(conds, cur, nil),
				"if this assertion flipped, array path support shipped - update the test to assert real semantics")
		})
	}
}

// TestMatchOperatorEdgeCasesOnCRD covers operator-specific corner cases that
// arise when matching on a custom resource: missing old object, missing field
// before vs after, and the changed_from-on-deletion case the existing matcher
// special-cases.
func TestMatchOperatorEdgeCasesOnCRD(t *testing.T) {
	withCondition := argoRollout("Healthy", float64(3), false)

	noPhase := argoRollout("", float64(3), false)
	delete(noPhase["status"].(map[string]interface{}), "phase")

	tests := map[string]struct {
		path     string
		op       v1.WorkflowTriggerFieldOperator
		value    string
		obj      any
		oldObj   any
		expected bool
	}{
		"changed without old object is false": {
			".status.phase", v1.FieldOperatorChanged, "", withCondition, nil, false,
		},
		"changed_to without old object is false": {
			".status.phase", v1.FieldOperatorChangedTo, "Healthy", withCondition, nil, false,
		},
		"changed_to when field appears for the first time": {
			".status.phase", v1.FieldOperatorChangedTo, "Healthy", withCondition, noPhase, true,
		},
		"changed_from when field is removed": {
			".status.phase", v1.FieldOperatorChangedFrom, "Healthy", noPhase, withCondition, true,
		},
		"equals on missing field is false": {
			".status.health", v1.FieldOperatorEquals, "ok", withCondition, nil, false,
		},
		"not_equals on missing field is false": {
			// Field doesn't exist → no resolved value → not_equals can't be true.
			".status.health", v1.FieldOperatorNotEquals, "ok", withCondition, nil, false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			conds := []v1.WorkflowTriggerFieldCondition{{Path: tc.path, Operator: tc.op, Value: tc.value}}
			assert.Equal(t, tc.expected, matchFieldSelector(conds, tc.obj, tc.oldObj))
		})
	}
}

// TestMatchAndAggregationOnCRD covers the WorkflowTriggerFieldCondition slice
// being AND-aggregated - one false condition fails the whole match.
func TestMatchAndAggregationOnCRD(t *testing.T) {
	cur := argoRollout("Healthy", float64(5), true)

	allMatch := []v1.WorkflowTriggerFieldCondition{
		{Path: ".status.phase", Operator: v1.FieldOperatorEquals, Value: "Healthy"},
		{Path: ".spec.paused", Operator: v1.FieldOperatorEquals, Value: "true"},
		{Path: ".spec.replicas", Operator: v1.FieldOperatorEquals, Value: "5"},
	}
	assert.True(t, matchFieldSelector(allMatch, cur, nil), "all conditions hold")

	oneMissing := append([]v1.WorkflowTriggerFieldCondition{}, allMatch...)
	oneMissing = append(oneMissing, v1.WorkflowTriggerFieldCondition{
		Path: ".status.phase", Operator: v1.FieldOperatorEquals, Value: "Degraded",
	})
	assert.False(t, matchFieldSelector(oneMissing, cur, nil), "one wrong condition fails the AND")
}
