package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateLicense(t *testing.T) {
	// Test for successful validation
	t.Run("Success", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			var reqBody LicenseRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			if err != nil || reqBody.License != "valid-license" {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			response := LicenseResponse{Valid: true, Message: "License is valid"}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			// Corrected the response struct to match the expected fields
		}))
		defer mockServer.Close()

		client := NewClient().WithURL(mockServer.URL)

		resp, err := client.ValidateLicense(LicenseRequest{License: "valid-license"})
		assert.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Equal(t, "License is valid", resp.Message)
	})

	// Test for failure due to invalid request
	t.Run("FailureInvalidRequest", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		}))
		defer mockServer.Close()

		client := NewClient().WithURL(mockServer.URL)

		resp, err := client.ValidateLicense(LicenseRequest{License: "invalid-license"})
		assert.NoError(t, err)
		assert.False(t, resp.Valid)
	})
}
