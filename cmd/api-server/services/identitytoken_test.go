package services

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/identitytoken"
)

// Cloud providers fetch the issuer documents from the API root with a plain GET.
func TestRegisterIdentityTokenRoutes(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := identitytoken.NewSigner("https://testkube.example.com", key)
	require.NoError(t, err)
	app := fiber.New()
	RegisterIdentityTokenRoutes(app, signer)

	get := func(t *testing.T, path string) (*http.Response, map[string]any) {
		t.Helper()
		resp, err := app.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		var doc map[string]any
		if resp.StatusCode == http.StatusOK {
			require.NoError(t, json.Unmarshal(body, &doc))
		}
		return resp, doc
	}

	resp, doc := get(t, identitytoken.DiscoveryPath)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "public, max-age=300", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "https://testkube.example.com", doc["issuer"])
	assert.Equal(t, "https://testkube.example.com"+identitytoken.JWKSPath, doc["jwks_uri"])

	resp, doc = get(t, identitytoken.JWKSPath)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Len(t, doc["keys"], 1)

	resp, _ = get(t, identitytoken.JWKSPath+"/extra")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
