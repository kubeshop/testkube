package v1

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflow "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// The duration bounds come from the query string; a value that is not a positive number is ignored,
// like the other numeric filters, so a typo widens the list instead of failing the request.
func TestGetWorkflowExecutionsFilterFromRequest_Duration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		query   string
		wantMin int64
		wantMax int64
	}{
		{"", 0, 0},
		{"minDurationMs=1000", 1000, 0},
		{"maxDurationMs=5000", 0, 5000},
		{"minDurationMs=1000&maxDurationMs=5000", 1000, 5000},
		{"minDurationMs=abc&maxDurationMs=-5", 0, 0},
		{"minDurationMs=0", 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			var filter testworkflow.Filter
			app := fiber.New()
			app.Get("/*", func(c *fiber.Ctx) error {
				filter = getWorkflowExecutionsFilterFromRequest(c)
				return nil
			})

			_, err := app.Test(httptest.NewRequest("GET", "/test-workflow-executions?"+tc.query, nil))
			require.NoError(t, err)

			assert.Equal(t, tc.wantMin, filter.MinDurationMs(), "got min %d for %q, want %d", filter.MinDurationMs(), tc.query, tc.wantMin)
			assert.Equal(t, tc.wantMax, filter.MaxDurationMs(), "got max %d for %q, want %d", filter.MaxDurationMs(), tc.query, tc.wantMax)
		})
	}
}
