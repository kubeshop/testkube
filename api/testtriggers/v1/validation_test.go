package v1

import (
	"testing"

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
