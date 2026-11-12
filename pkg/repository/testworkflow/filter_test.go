package testworkflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Zero means "no bound", so a filter with neither bound set adds nothing to the query.
func TestFilterImpl_DurationDefined(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter *FilterImpl
		want   bool
	}{
		{"no bounds", NewExecutionsFilter(), false},
		{"minimum only", NewExecutionsFilter().WithMinDurationMs(1_000), true},
		{"maximum only", NewExecutionsFilter().WithMaxDurationMs(5_000), true},
		{"both", NewExecutionsFilter().WithMinDurationMs(1_000).WithMaxDurationMs(5_000), true},
		{"zero is no bound", NewExecutionsFilter().WithMinDurationMs(0).WithMaxDurationMs(0), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.DurationDefined(), "got DurationDefined %v for %s, want %v", tc.filter.DurationDefined(), tc.name, tc.want)
		})
	}
}
