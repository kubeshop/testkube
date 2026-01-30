package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// TestAbortFilterType ensures the filter created in abortAllTestWorkflowExecutionsHandlerStandalone
// is compatible with the cloud repository's type assertion requirement.
// This test prevents the panic: interface conversion: testworkflow.Filter is testworkflow.FilterImpl, not *testworkflow.FilterImpl
func TestAbortFilterType(t *testing.T) {
	// Create a filter using the same pattern as abortAllTestWorkflowExecutionsHandlerStandalone
	filter := testworkflow2.NewExecutionsFilter().WithName("test-workflow")
	
	// Verify that the filter is a pointer type
	assert.IsType(t, &testworkflow2.FilterImpl{}, filter, "Filter should be a pointer to FilterImpl")
	
	// Verify that type assertion to *FilterImpl succeeds (this would panic if it's a value type)
	var iface testworkflow2.Filter = filter
	_, ok := iface.(*testworkflow2.FilterImpl)
	assert.True(t, ok, "Filter should be assertable to *FilterImpl")
}
