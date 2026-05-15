package v1

import "fmt"

// Validate checks the WorkflowTriggerSpec for logical errors that can't be
// caught by CRD schema validation alone. Called from both the REST API and
// the CRD sync handler.
func (s *WorkflowTriggerSpec) Validate() []error {
	var errs []error

	// watch is required when event is set
	if s.When.Event != "" && s.Watch == nil && s.When.Git == nil {
		errs = append(errs, fmt.Errorf("watch or when.git is required when when.event is set"))
	}

	// watch.resource.kind is required
	if s.Watch != nil && s.Watch.Resource.Kind == "" {
		errs = append(errs, fmt.Errorf("watch.resource.kind is required"))
	}

	// at least one trigger source must be set
	if s.When.Event == "" {
		errs = append(errs, fmt.Errorf("when.event is required"))
	}

	// git triggers currently support only modified events and must not set watch,
	// because watch is interpreted as a Kubernetes resource informer target.
	if s.When.Git != nil && s.When.Event != "" && s.When.Event != "modified" {
		errs = append(errs, fmt.Errorf("when.event must be \"modified\" when when.git is set"))
	}
	if s.When.Git != nil && s.When.Git.Uri == "" {
		errs = append(errs, fmt.Errorf("when.git.uri is required when when.git is set"))
	}
	if s.When.Git != nil && s.Watch != nil {
		errs = append(errs, fmt.Errorf("watch must be omitted when when.git is set"))
	}
	if s.When.Git != nil && s.Match != nil {
		errs = append(errs, fmt.Errorf("match is not supported when when.git is set"))
	}
	if s.When.Git != nil && s.Wait != nil {
		errs = append(errs, fmt.Errorf("wait is not supported when when.git is set"))
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
