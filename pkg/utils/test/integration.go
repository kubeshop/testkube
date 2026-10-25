package test

import (
	"os"
	"testing"
)

func IntegrationTest(t testing.TB) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration tests because environment variable INTEGRATION is not set")
	}
}
