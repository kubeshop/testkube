package ui

import (
	"os"
	"testing"
)

func promptUnderTest() string { return os.Getenv("TESTKUBE_UI_PROMPT") }

func testBinary(t *testing.T) string {
	t.Helper()

	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot locate the test binary: %v", err)
	}
	return binary
}
