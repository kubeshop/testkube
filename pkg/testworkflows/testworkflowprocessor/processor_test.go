package testworkflowprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type dummyInspector struct{}

func (*dummyInspector) Inspect(_ context.Context, _, _ string, _ corev1.PullPolicy, _ []string) (*imageinspector.Info, error) {
	return &imageinspector.Info{
		Entrypoint: []string{"/bin/sh"},
		Cmd:        []string{"-c"},
		User:       0,
		Group:      1001,
	}, nil
}

func (*dummyInspector) ResolveName(_, image string) string {
	return image
}

func TestBundle_InvalidEmptyDirSizeLimit_ReturnsError(t *testing.T) {
	proc := New(&dummyInspector{})
	workflow := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				EmptyDirSizeLimit: "not-a-quantity",
			},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, `invalid worker emptyDir sizeLimit "not-a-quantity"`)
}

func TestBundle_InvalidDefaultImagePullPolicy_ReturnsError(t *testing.T) {
	proc := New(&dummyInspector{})
	workflow := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				DefaultImagePullPolicy: "InvalidPolicy",
			},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, `invalid worker default image pull policy "InvalidPolicy"`)
}

func TestBundle_InvalidDefaultRunnerResources_ReturnsError(t *testing.T) {
	proc := New(&dummyInspector{})
	workflow := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				DefaultRunnerResources: testworkflowconfig.ContainerResourceConfig{
					Requests: testworkflowconfig.ContainerResources{
						CPU: "not-valid",
					},
				},
			},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, `invalid worker default runner CPU request "not-valid"`)
}

func TestBundle_DefaultImagePullPolicy_Applied(t *testing.T) {
	proc := New(&dummyInspector{}).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessNestedSteps)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}},
			},
		},
	}

	bundle, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				DefaultImagePullPolicy: "Always",
			},
		},
	})

	require.NoError(t, err)
	allContainers := append(bundle.Job.Spec.Template.Spec.InitContainers, bundle.Job.Spec.Template.Spec.Containers...)
	for _, c := range allContainers {
		assert.Equalf(t, corev1.PullAlways, c.ImagePullPolicy, "container %s should have PullAlways", c.Name)
	}
}

func TestBundle_DefaultRunnerResources_Applied(t *testing.T) {
	proc := New(&dummyInspector{}).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessNestedSteps)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}},
			},
		},
	}

	bundle, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				DefaultRunnerResources: testworkflowconfig.ContainerResourceConfig{
					Requests: testworkflowconfig.ContainerResources{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Limits: testworkflowconfig.ContainerResources{
						CPU:    "500m",
						Memory: "512Mi",
					},
				},
			},
		},
	})

	require.NoError(t, err)
	allContainers := append(bundle.Job.Spec.Template.Spec.InitContainers, bundle.Job.Spec.Template.Spec.Containers...)
	for _, c := range allContainers {
		assert.Equal(t, resource.MustParse("100m"), c.Resources.Requests[corev1.ResourceCPU], "container %s CPU request", c.Name)
		assert.Equal(t, resource.MustParse("128Mi"), c.Resources.Requests[corev1.ResourceMemory], "container %s memory request", c.Name)
		assert.Equal(t, resource.MustParse("500m"), c.Resources.Limits[corev1.ResourceCPU], "container %s CPU limit", c.Name)
		assert.Equal(t, resource.MustParse("512Mi"), c.Resources.Limits[corev1.ResourceMemory], "container %s memory limit", c.Name)
	}
}
