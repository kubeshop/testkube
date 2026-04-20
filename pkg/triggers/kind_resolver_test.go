package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBuiltinResource(t *testing.T) {
	tests := map[string]struct {
		kind     string
		expected bool
	}{
		"deployment lowercase":   {kind: "deployment", expected: true},
		"Deployment capitalized": {kind: "Deployment", expected: true},
		"DEPLOYMENT uppercase":   {kind: "DEPLOYMENT", expected: true},
		"pod":                    {kind: "pod", expected: true},
		"Pod":                    {kind: "Pod", expected: true},
		"statefulset":            {kind: "statefulset", expected: true},
		"daemonset":              {kind: "daemonset", expected: true},
		"service":                {kind: "service", expected: true},
		"ingress":                {kind: "ingress", expected: true},
		"event":                  {kind: "event", expected: true},
		"configmap":              {kind: "configmap", expected: true},
		"ConfigMap":              {kind: "ConfigMap", expected: true},
		"Rollout":                {kind: "Rollout", expected: false},
		"KafkaTopic":             {kind: "KafkaTopic", expected: false},
		"Certificate":            {kind: "Certificate", expected: false},
		"VirtualService":         {kind: "VirtualService", expected: false},
		"empty":                  {kind: "", expected: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isBuiltinResource(tc.kind))
		})
	}
}
