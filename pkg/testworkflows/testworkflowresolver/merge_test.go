package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestMergePodConfig_Resources(t *testing.T) {
	tests := []struct {
		name    string
		dst     *testworkflowsv1.Resources
		include *testworkflowsv1.Resources
		want    *testworkflowsv1.Resources
	}{
		{
			name: "nil include keeps dst",
			dst: &testworkflowsv1.Resources{
				Limits: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceMemory: "512Mi"},
			},
			want: &testworkflowsv1.Resources{
				Limits: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceMemory: "512Mi"},
			},
		},
		{
			name: "nil dst takes include",
			include: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "200m"},
			},
			want: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "200m"},
			},
		},
		{
			name: "include overrides the same key and keeps the other keys",
			dst: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "100m", corev1.ResourceMemory: "128Mi"},
				Limits:   map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "1", corev1.ResourceMemory: "256Mi"},
			},
			include: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "200m"},
				Limits:   map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceMemory: "512Mi"},
			},
			want: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "200m", corev1.ResourceMemory: "128Mi"},
				Limits:   map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "1", corev1.ResourceMemory: "512Mi"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergePodConfig(
				&testworkflowsv1.PodConfig{Resources: tt.dst},
				&testworkflowsv1.PodConfig{Resources: tt.include},
			)
			assert.Equal(t, tt.want, result.Resources)
		})
	}
}
