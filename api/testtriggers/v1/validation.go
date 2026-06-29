package v1

import (
	"fmt"
	"regexp"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// MatchPathPattern is the canonical regex for match-condition paths. Exported
// so cp-api and the UI can pin to the same shape. Accepts dot-separated
// identifiers, with optional `[*]` or `[N]` suffixes per segment for forward
// compatibility with array-aware backends.
var MatchPathPattern = regexp.MustCompile(`^\.[A-Za-z0-9_-]+(\[\*\]|\[\d+\])?(\.[A-Za-z0-9_-]+(\[\*\]|\[\d+\])?)*$`)

// MatchPathBracketPattern detects array-segment suffixes (`[*]` or `[N]`).
// The matcher's expression engine can't tokenize either today, so paths with
// bracket suffixes silently no-op at fire time. We reject them at save time
// to surface the gap loudly. Lift this once pkg/expressions gains support.
var MatchPathBracketPattern = regexp.MustCompile(`\[\*\]|\[\d+\]`)

// MatchReason* are reason codes carried on MatchValidationError so callers can
// categorize a match[] rejection without parsing the message. The strings are
// a stable contract (e.g. cp-api labels Prometheus counters with them).
const (
	MatchReasonPathRequired  = "path_required"
	MatchReasonBadPathSyntax = "bad_path_syntax"
	MatchReasonArrayPath     = "array_path_unsupported"
	MatchReasonValueRequired = "value_required"
	MatchReasonUnknownOp     = "unknown_op"
	MatchReasonEventMismatch = "event_mismatch"
)

// MatchValidationError is a single match[] validation failure tagged with a
// Reason (one of the MatchReason* codes). It satisfies error, so callers that
// only check len(errs) > 0 are unaffected.
type MatchValidationError struct {
	Reason  string
	Message string
}

func (e *MatchValidationError) Error() string { return e.Message }

func matchErr(reason, format string, args ...any) *MatchValidationError {
	return &MatchValidationError{Reason: reason, Message: fmt.Sprintf(format, args...)}
}

// ValidateMatchConditions validates the per-condition match[] rules: path
// syntax, the operator/value contract, and the change-operator/modified-event
// coupling. It returns a MatchValidationError per failure. The listener-binding
// and resource-compatibility rules live in Validate; this is the slice cp-api
// reuses (via the same package) so the two implementations can't drift.
func ValidateMatchConditions(conditions []workflowtriggersv1.WorkflowTriggerFieldCondition, event TestTriggerEvent) []error {
	var errs []error
	for i, cond := range conditions {
		if cond.Path == "" {
			errs = append(errs, matchErr(MatchReasonPathRequired, "match[%d].path is required", i))
			continue
		}
		if !MatchPathPattern.MatchString(cond.Path) {
			errs = append(errs, matchErr(MatchReasonBadPathSyntax, "match[%d].path %q is not a valid dot-path (e.g. .status.phase)", i, cond.Path))
		}
		if MatchPathBracketPattern.MatchString(cond.Path) {
			errs = append(errs, matchErr(MatchReasonArrayPath, "match[%d].path %q contains an array index/wildcard ([*] or [N]); array-path matching is not supported yet, please match on a scalar field", i, cond.Path))
		}

		switch cond.Operator {
		case workflowtriggersv1.FieldOperatorEquals,
			workflowtriggersv1.FieldOperatorNotEquals,
			workflowtriggersv1.FieldOperatorChangedTo,
			workflowtriggersv1.FieldOperatorChangedFrom:
			if cond.Value == "" {
				errs = append(errs, matchErr(MatchReasonValueRequired, "match[%d]: operator %q requires a value", i, cond.Operator))
			}
		case workflowtriggersv1.FieldOperatorExists,
			workflowtriggersv1.FieldOperatorNotExists,
			workflowtriggersv1.FieldOperatorChanged:
			// no value needed
		default:
			errs = append(errs, matchErr(MatchReasonUnknownOp, "match[%d]: unknown operator %q", i, cond.Operator))
		}

		// change-based operators require the modified event
		switch cond.Operator {
		case workflowtriggersv1.FieldOperatorChanged,
			workflowtriggersv1.FieldOperatorChangedTo,
			workflowtriggersv1.FieldOperatorChangedFrom:
			if event != TestTriggerEventModified {
				errs = append(errs, matchErr(MatchReasonEventMismatch, "match[%d]: operator %q requires event to be %q", i, cond.Operator, TestTriggerEventModified))
			}
		}
	}
	return errs
}

// Validate checks the TestTriggerSpec for logical errors that can't be caught
// by CRD schema validation alone. It is the single validation gate: the REST
// create, update, and bulk-update handlers all round-trip the request through
// the CRD mapper and call this, as does the admission webhook (when wired).
//
// Schema-aware match[] triggers require listener binding via
// listenerAgentIds. The listener is the agent that watches the cluster and
// evaluates match[] at fire time, so validation against a CRD schema is only
// sound when that listener is known at save time; broadcast dispatch would
// let the trigger land on a listener whose schema or RBAC differs from the
// one validation ran against.
func (s *TestTriggerSpec) Validate() []error {
	var errs []error

	isContentResource := s.Resource == TestTriggerResourceContent || (s.ResourceRef != nil && s.ResourceRef.Kind == string(TestTriggerResourceContent))

	if isContentResource && s.Event != TestTriggerEventGitPush && s.Event != TestTriggerEventGitTagPush && s.Event != TestTriggerEventGitPullRequest {
		errs = append(errs, fmt.Errorf("resource %q requires event to be one of %q, %q, or %q", TestTriggerResourceContent, TestTriggerEventGitPush, TestTriggerEventGitTagPush, TestTriggerEventGitPullRequest))
	}
	if isContentResource && s.ConditionSpec != nil && len(s.ConditionSpec.Conditions) > 0 {
		errs = append(errs, fmt.Errorf("resource %q does not support conditionSpec.conditions", TestTriggerResourceContent))
	}
	if isContentResource && s.ProbeSpec != nil && len(s.ProbeSpec.Probes) > 0 {
		errs = append(errs, fmt.Errorf("resource %q does not support probeSpec.probes", TestTriggerResourceContent))
	}
	if isContentResource && (s.ContentSelector == nil || s.ContentSelector.Git == nil || s.ContentSelector.Git.Uri == "") {
		errs = append(errs, fmt.Errorf("resource %q requires contentSelector.git.uri", TestTriggerResourceContent))
	}
	if isContentResource && len(s.Match) > 0 {
		errs = append(errs, fmt.Errorf("resource %q does not support match", TestTriggerResourceContent))
	} else if len(s.Match) > 0 {
		if s.Listener == nil || len(s.Listener.Match["id"]) == 0 {
			errs = append(errs, fmt.Errorf("match conditions require listener.match.id to pin the trigger to one or more listener agents"))
		}
	}

	errs = append(errs, ValidateMatchConditions(s.Match, s.Event)...)

	return errs
}
