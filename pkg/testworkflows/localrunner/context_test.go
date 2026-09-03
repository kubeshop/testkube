package localrunner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalContext(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{name: "kind-local", want: true},
		{name: "k3d-local", want: true},
		{name: "production", want: false},
		{name: "Kind-local", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalContext(tt.name); got != tt.want {
				t.Fatalf("isLocalContext(%q) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	for _, namespace := range []string{"", "bad/name", "bad\\name", "bad namespace", "bad;namespace", "UPPERCASE"} {
		if err := ValidateNamespace(namespace); err == nil {
			t.Fatalf("ValidateNamespace(%q) unexpectedly succeeded", namespace)
		}
	}
	if err := ValidateNamespace(DefaultNamespace); err != nil {
		t.Fatalf("ValidateNamespace(%q): %v", DefaultNamespace, err)
	}
}

func TestRecoveryHintsQuoteDynamicArgumentsAndPreserveKubeconfig(t *testing.T) {
	const (
		runID      = "local-demo"
		namespace  = "testkube-local"
		context    = "kind-demo; echo unsafe"
		kubeconfig = "/tmp/local cluster's config"
	)
	inspect := localInspectHint(runID, namespace, context, kubeconfig)
	clean := localCleanHint(runID, namespace, context, kubeconfig)
	assert.Contains(t, inspect, "--kubeconfig '/tmp/local cluster'\"'\"'s config'")
	assert.Contains(t, inspect, "--context 'kind-demo; echo unsafe'")
	assert.NotContains(t, inspect, "--context kind-demo;")
	assert.Contains(t, clean, "--kubeconfig '/tmp/local cluster'\"'\"'s config'")
	assert.Contains(t, clean, "--context 'kind-demo; echo unsafe'")
	assert.Contains(t, clean, "'local-demo'")
}

func TestInspectHintUsesNormalizedLongRunIDLabel(t *testing.T) {
	runID := "local-inspect-" + strings.Repeat("a", 80)
	labelRunID, err := localRunIDLabelValue(runID)
	if err != nil {
		t.Fatalf("localRunIDLabelValue(%q): %v", runID, err)
	}

	inspect := localInspectHint(runID, DefaultNamespace, "kind-local", "")
	assert.Contains(t, inspect, LocalRunIDLabel+"='"+labelRunID+"'")
	assert.NotContains(t, inspect, runID)
}
