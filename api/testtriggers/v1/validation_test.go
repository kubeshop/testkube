package v1

import "testing"

func TestTestTriggerSpecValidate_ContentRequiresModifiedEvent(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventCreated,
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
		Event:    TestTriggerEventModified,
	}

	errs := spec.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %d", len(errs))
	}
}
