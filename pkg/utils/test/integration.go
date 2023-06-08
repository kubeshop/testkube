package test

import (
	"os"
	"testing"
)

func IntegrationTest(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration tests because environment variable INTEGRATION is not set")
	}
}
