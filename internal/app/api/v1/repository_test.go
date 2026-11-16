package v1

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Proxies may replace a 5xx body and lose the reason.
func TestValidateRepositoryHandlerReportsAFailedCheckAsBadRequest(t *testing.T) {
	git := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(git.Close)
	app := fiber.New()
	app.Post("/repositories", (&TestkubeAPI{}).ValidateRepositoryHandler())

	body := `{"type":"git","uri":"` + git.URL + `/repo.git"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/repositories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	reply, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(reply), git.URL+"/repo.git/' not found")
}
