package v1

import (
	"fmt"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// Validate checks the TestTriggerSpec for logical errors that can't be caught
// by CRD schema validation alone. Called from REST create/update handlers.
// Currently scoped to the match[] field, mirroring WorkflowTriggerSpec.Validate.
func (s *TestTriggerSpec) Validate() []error {
	var errs []error

	isContentResource := s.Resource == TestTriggerResourceContent || (s.ResourceRef != nil && s.ResourceRef.Kind == string(TestTriggerResourceContent))

	if isContentResource && s.Event != TestTriggerEventGitPush && s.Event != TestTriggerEventGitTagPush && s.Event != TestTriggerEventGitPullRequest {
		errs = append(errs, fmt.Errorf("resource %q requires event to be %q", TestTriggerResourceContent, TestTriggerEventModified))
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
	}

	for i, cond := range s.Match {
		if cond.Path == "" {
			errs = append(errs, fmt.Errorf("match[%d].path is required", i))
			continue
		}

		switch cond.Operator {
		case workflowtriggersv1.FieldOperatorEquals,
			workflowtriggersv1.FieldOperatorNotEquals,
			workflowtriggersv1.FieldOperatorChangedTo,
			workflowtriggersv1.FieldOperatorChangedFrom:
			if cond.Value == "" {
				errs = append(errs, fmt.Errorf("match[%d]: operator %q requires a value", i, cond.Operator))
			}
		case workflowtriggersv1.FieldOperatorExists,
			workflowtriggersv1.FieldOperatorNotExists,
			workflowtriggersv1.FieldOperatorChanged:
			// no value needed
		default:
			errs = append(errs, fmt.Errorf("match[%d]: unknown operator %q", i, cond.Operator))
		}

		// change-based operators require the modified event
		switch cond.Operator {
		case workflowtriggersv1.FieldOperatorChanged,
			workflowtriggersv1.FieldOperatorChangedTo,
			workflowtriggersv1.FieldOperatorChangedFrom:
			if s.Event != "" && s.Event != TestTriggerEventModified {
				errs = append(errs, fmt.Errorf("match[%d]: operator %q requires event to be %q", i, cond.Operator, TestTriggerEventModified))
			}
		}
	}

	return errs
}
