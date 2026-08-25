package testkube

// Friendly aliases for swagger-codegen constants. Kept in an extended file so
// they survive OpenAPI model regeneration.
const (
	TestTriggerFieldOperatorEquals      = EQUALS_TestTriggerFieldOperator
	TestTriggerFieldOperatorNotEquals   = NOT_EQUALS_TestTriggerFieldOperator
	TestTriggerFieldOperatorExists      = EXISTS_TestTriggerFieldOperator
	TestTriggerFieldOperatorNotExists   = NOT_EXISTS_TestTriggerFieldOperator
	TestTriggerFieldOperatorChanged     = CHANGED_TestTriggerFieldOperator
	TestTriggerFieldOperatorChangedTo   = CHANGED_TO_TestTriggerFieldOperator
	TestTriggerFieldOperatorChangedFrom = CHANGED_FROM_TestTriggerFieldOperator
)
