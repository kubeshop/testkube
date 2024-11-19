package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

		response, err := client.ValidateLicense(LicenseRequest{License: "valid-license"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !response.Valid {
			t.Errorf("Expected license to be valid, got %v", response.Valid)
		}
		if response.Message != "License is valid" {
			t.Errorf("Expected message 'License is valid', got %s", response.Message)
		}
	})

	// Test for failure due to invalid request
	t.Run("FailureInvalidRequest", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		}))
		defer mockServer.Close()

		client := NewClient().WithURL(mockServer.URL)

		_, err := client.ValidateLicense(LicenseRequest{License: "invalid-license"})
		if err == nil {
			t.Errorf("Expected error due to invalid request, got none")
		}
	})

	t.Run("RealValidation license valid", func(t *testing.T) {
		client := NewClient()

		response, err := client.ValidateLicense(LicenseRequest{License: "AB24F3-405E39-C3F657-94D113-F06C13-V3"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !response.Valid {
			t.Errorf("Expected license to be valid, got %v", response.Valid)
		}
		if response.Code != "VALID" {
			t.Errorf("Expected message 'VALID', got %s", response.Code)
		}
	})
}
