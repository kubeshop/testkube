package leasebackend

import (
	"strings"
	"testing"
)

func TestSanitizeForK8sName_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "already valid", input: "my-cluster", expected: "my-cluster"},
		{name: "uppercase", input: "My-Cluster", expected: "my-cluster"},
		{name: "underscores", input: "tkcroot_abc", expected: "tkcroot-abc"},
		{name: "dots replaced", input: "my.cluster.id", expected: "my-cluster-id"},
		{name: "mixed invalid", input: "a_b.c!d@e", expected: "a-b-c-d-e"},
		{name: "leading hyphens trimmed", input: "---valid", expected: "valid"},
		{name: "trailing hyphens trimmed", input: "valid---", expected: "valid"},
		{name: "both trimmed", input: "---valid---", expected: "valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForK8sName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeForK8sName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeForK8sName_Truncation(t *testing.T) {
	// 70-char input that is already valid lowercase
	long := strings.Repeat("a", 70)
	got := SanitizeForK8sName(long)
	if len(got) > 63 {
		t.Errorf("expected len <= 63, got %d: %q", len(got), got)
	}
	// Should end with a hash suffix
	if !strings.Contains(got, "-") {
		t.Errorf("expected hash-suffix separator, got %q", got)
	}
}

func TestSanitizeForK8sName_TruncationPreservesUniqueness(t *testing.T) {
	// Two inputs that share a long prefix but differ at the end
	a := strings.Repeat("a", 60) + "-agent-1"
	b := strings.Repeat("a", 60) + "-agent-2"
	gotA := SanitizeForK8sName(a)
	gotB := SanitizeForK8sName(b)
	if gotA == gotB {
		t.Errorf("distinct inputs should produce distinct names, both got %q", gotA)
	}
	if len(gotA) > 63 || len(gotB) > 63 {
		t.Errorf("names exceed 63 chars: %q (%d), %q (%d)", gotA, len(gotA), gotB, len(gotB))
	}
}

func TestSanitizeForK8sName_EmptyFallback(t *testing.T) {
	// All-invalid characters should produce a deterministic hash
	got := SanitizeForK8sName("___")
	if got == "" {
		t.Fatal("expected non-empty fallback for all-invalid input")
	}
	if len(got) != 8 {
		t.Errorf("expected 8-char hash fallback, got %q (len %d)", got, len(got))
	}
	// Deterministic
	if got2 := SanitizeForK8sName("___"); got != got2 {
		t.Errorf("expected deterministic fallback, got %q and %q", got, got2)
	}
}

func TestSanitizeForK8sName_DistinctAllInvalid(t *testing.T) {
	// Distinct all-invalid inputs should produce distinct fallbacks
	a := SanitizeForK8sName("___")
	b := SanitizeForK8sName("!!!")
	if a == b {
		t.Errorf("distinct all-invalid inputs should produce distinct fallbacks, both got %q", a)
	}
}

func TestSanitizeForK8sName_EmptyInput(t *testing.T) {
	got := SanitizeForK8sName("")
	if got == "" {
		t.Fatal("expected non-empty fallback for empty input")
	}
	if len(got) != 8 {
		t.Errorf("expected 8-char hash fallback, got %q (len %d)", got, len(got))
	}
}

func TestSanitizeForK8sName_TruncationEndsAlphanumeric(t *testing.T) {
	// Construct input where char at position 54 is a hyphen after sanitization
	// e.g., 53 'a' + '_' + ... = hyphen at position 54
	input := strings.Repeat("a", 53) + "_" + strings.Repeat("b", 20)
	got := SanitizeForK8sName(input)
	if len(got) > 63 {
		t.Errorf("expected len <= 63, got %d", len(got))
	}
	// Must not end with hyphen (DNS-1123)
	if got[len(got)-1] == '-' {
		t.Errorf("name must not end with hyphen: %q", got)
	}
	// Must not start with hyphen
	if got[0] == '-' {
		t.Errorf("name must not start with hyphen: %q", got)
	}
}
