package v1

import (
	"testing"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

func TestTestTriggerSpecValidate_ContentRequiresModifiedEvent(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventCreated,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resource with non-modified event")
	}
}

func TestTestTriggerSpecValidate_ContentWithModifiedEvent(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventGitPush,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %d", len(errs))
	}
}

func TestTestTriggerSpecValidate_ContentRejectsConditionSpecConditions(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventGitPush,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
		ConditionSpec: &TestTriggerConditionSpec{
			Conditions: []TestTriggerCondition{
				{Type_: "Ready"},
			},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resource with conditionSpec.conditions")
	}
}

func TestTestTriggerSpecValidate_ContentResourceRefRequiresModifiedEvent(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		ResourceRef: &TestTriggerResourceRef{Kind: "content"},
		Event:       TestTriggerEventCreated,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resourceRef with non-modified event")
	}
}

func TestTestTriggerSpecValidate_ContentResourceRefRejectsConditionSpecConditions(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		ResourceRef: &TestTriggerResourceRef{Kind: "content"},
		Event:       TestTriggerEventGitPush,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
		ConditionSpec: &TestTriggerConditionSpec{
			Conditions: []TestTriggerCondition{
				{Type_: "Ready"},
			},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resourceRef with conditionSpec.conditions")
	}
}

func TestTestTriggerSpecValidate_ContentRejectsMatch(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventGitPush,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
		Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
			{Path: ".metadata.name", Operator: workflowtriggersv1.FieldOperatorExists},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resource with match")
	}
}

func TestTestTriggerSpecValidate_ContentWithGitPullRequestEvent(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventGitPullRequest,
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for git-pull-request event, got %d: %v", len(errs), errs)
	}
}

func TestTestTriggerSpecValidate_EventOnlyIsValid(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceDeployment,
		Event:    TestTriggerEventModified,
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for single event, got %d: %v", len(errs), errs)
	}
}

func TestTestTriggerSpecValidate_EventsOnlyIsValid(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceDeployment,
		Events:   []TestTriggerEvent{TestTriggerEventCreated, TestTriggerEventModified},
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for events list, got %d: %v", len(errs), errs)
	}
}

func TestTestTriggerSpecValidate_RejectsBothEventAndEvents(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceDeployment,
		Event:    TestTriggerEventModified,
		Events:   []TestTriggerEvent{TestTriggerEventCreated},
	}

	errs := spec.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly one validation error when both event and events are set, got %d: %v", len(errs), errs)
	}
	want := "only one of event or events can be set"
	if errs[0].Error() != want {
		t.Fatalf("expected error %q, got %q", want, errs[0].Error())
	}
}

func TestTestTriggerSpecValidate_RejectsNeitherEventNorEvents(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceDeployment,
	}

	errs := spec.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly one validation error when neither event nor events is set, got %d: %v", len(errs), errs)
	}
	want := "one of event or events must be set"
	if errs[0].Error() != want {
		t.Fatalf("expected error %q, got %q", want, errs[0].Error())
	}
}

func TestTestTriggerSpecValidate_ContentWithGitEventsList(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Events:   []TestTriggerEvent{TestTriggerEventGitPush, TestTriggerEventGitTagPush},
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for git events list, got %d: %v", len(errs), errs)
	}
}

func TestTestTriggerSpecValidate_ContentRejectsNonGitEventInEventsList(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Events:   []TestTriggerEvent{TestTriggerEventGitPush, TestTriggerEventModified},
		ContentSelector: &TestTriggerContentSelector{
			Git: &TestTriggerContentGitSpec{
				Uri: "https://github.com/kubeshop/testkube",
			},
		},
	}

	errs := spec.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected validation error for content resource with a non-git event in events list")
	}
}

func TestTestTriggerSpecValidate_MatchChangeOperatorRequiresModifiedForEveryEvent(t *testing.T) {
	t.Parallel()

	match := []workflowtriggersv1.WorkflowTriggerFieldCondition{
		{Path: ".status.phase", Operator: workflowtriggersv1.FieldOperatorChanged},
	}
	listener := &commonv1.Target{Match: map[string][]string{"id": {"agent-1"}}}

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceDeployment,
		Events:   []TestTriggerEvent{TestTriggerEventModified},
		Match:    match,
		Listener: listener,
	}
	if errs := spec.Validate(); len(errs) != 0 {
		t.Fatalf("expected no validation errors for change operator with events [modified], got %d: %v", len(errs), errs)
	}

	spec.Events = []TestTriggerEvent{TestTriggerEventCreated, TestTriggerEventModified}
	if errs := spec.Validate(); len(errs) != 1 {
		t.Fatalf("expected exactly one validation error for change operator with a non-modified event in events, got %d: %v", len(errs), errs)
	}
}
