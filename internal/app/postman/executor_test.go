package postman

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostmanExecutor_StartExecution(t *testing.T) {

	t.Run("runs newman command", func(t *testing.T) {

		executor := NewPostmanExecutor()

		body := strings.NewReader(`{"": ""}`)

		req := httptest.NewRequest("POST", "/v1/executions/", body)
		resp, err := executor.Mux.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode, 200)

	})

}
