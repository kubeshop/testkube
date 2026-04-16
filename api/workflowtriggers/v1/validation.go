package v1

import "fmt"

// Validate checks the WorkflowTriggerSpec for logical errors that can't be
// caught by CRD schema validation alone. Called from both the REST API and
// the CRD sync handler.
func (s *WorkflowTriggerSpec) Validate() []error {
	var errs []error

	// watch is required when event is set
	if s.When.Event != "" && s.Watch == nil {
		errs = append(errs, fmt.Errorf("watch is required when when.event is set"))
	}

	// watch.resource.kind is required
	if s.Watch != nil && s.Watch.Resource.Kind == "" {
		errs = append(errs, fmt.Errorf("watch.resource.kind is required"))
	}

	// at least one trigger source must be set
	if s.When.Event == "" {
		errs = append(errs, fmt.Errorf("when.event is required"))
	}

	// workflow selector must identify at least one workflow
	if s.Run.Workflow.Name == "" && s.Run.Workflow.NameRegex == "" && s.Run.Workflow.LabelSelector == nil {
		errs = append(errs, fmt.Errorf("run.workflow must specify name, nameRegex, or labelSelector"))
	}

	// validate match conditions
	for i, cond := range s.Match {
		if cond.Path == "" {
			errs = append(errs, fmt.Errorf("match[%d].path is required", i))
			continue
		}

		switch cond.Operator {
		case FieldOperatorEquals, FieldOperatorNotEquals, FieldOperatorChangedTo, FieldOperatorChangedFrom:
			if cond.Value == "" {
				errs = append(errs, fmt.Errorf("match[%d]: operator %q requires a value", i, cond.Operator))
			}
		case FieldOperatorExists, FieldOperatorNotExists, FieldOperatorChanged:
			// no value needed
		default:
			errs = append(errs, fmt.Errorf("match[%d]: unknown operator %q", i, cond.Operator))
		}

		// changed/changed_to/changed_from only work with modified events
		switch cond.Operator {
		case FieldOperatorChanged, FieldOperatorChangedTo, FieldOperatorChangedFrom:
			if s.When.Event != "" && s.When.Event != "modified" {
				errs = append(errs, fmt.Errorf("match[%d]: operator %q requires when.event to be \"modified\"", i, cond.Operator))
			}
		}
	}

	return errs
}
