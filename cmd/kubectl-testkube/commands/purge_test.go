package commands

import (
	"testing"
)

func TestShouldPurge(t *testing.T) {
	t.Run("skips the prompt when --yes is set", func(t *testing.T) {
		prompted := false
		ok := shouldPurge("testkube", "testkube", true, func(string) bool {
			prompted = true
			return false
		})

		if !ok {
			t.Fatal("expected purge to proceed when --yes is set")
		}
		if prompted {
			t.Fatal("expected no confirmation prompt when --yes is set")
		}
	})

	t.Run("stops when the confirmation is declined", func(t *testing.T) {
		if shouldPurge("testkube", "testkube", false, func(string) bool { return false }) {
			t.Fatal("expected purge to stop when the confirmation is declined")
		}
	})

	t.Run("proceeds when the confirmation is accepted", func(t *testing.T) {
		if !shouldPurge("testkube", "testkube", false, func(string) bool { return true }) {
			t.Fatal("expected purge to proceed when the confirmation is accepted")
		}
	})
}

func TestPurgeCmdHasYesFlag(t *testing.T) {
	flag := NewPurgeCmd().Flags().Lookup("yes")
	if flag == nil {
		t.Fatal("expected purge command to define a --yes flag")
	}
	if flag.Shorthand != "y" {
		t.Fatalf("expected -y shorthand, got %q", flag.Shorthand)
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected --yes to default to false, got %q", flag.DefValue)
	}
}
