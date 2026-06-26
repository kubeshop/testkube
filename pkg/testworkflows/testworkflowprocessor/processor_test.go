package testworkflowprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

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
		assert.Equalf(t, resource.MustParse("100m"), c.Resources.Requests[corev1.ResourceCPU], "container %s CPU request", c.Name)
		assert.Equalf(t, resource.MustParse("128Mi"), c.Resources.Requests[corev1.ResourceMemory], "container %s memory request", c.Name)
		assert.Equalf(t, resource.MustParse("500m"), c.Resources.Limits[corev1.ResourceCPU], "container %s CPU limit", c.Name)
		assert.Equalf(t, resource.MustParse("512Mi"), c.Resources.Limits[corev1.ResourceMemory], "container %s memory limit", c.Name)
	}
}

func TestBundle_DefaultImagePullPolicy_DoesNotOverrideExplicitPolicy(t *testing.T) {
	proc := New(&dummyInspector{}).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessNestedSteps)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Image:           "custom-image:latest",
								ImagePullPolicy: corev1.PullNever,
							},
						},
					},
				},
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
	// Find the container using the custom image; its explicit PullNever must not be overridden.
	found := false
	for _, c := range allContainers {
		if c.Image == "custom-image:latest" {
			found = true
			assert.Equalf(t, corev1.PullNever, c.ImagePullPolicy, "container %s should keep explicit PullNever policy", c.Name)
		}
	}
	require.True(t, found, "expected to find container with custom-image:latest")
}

func TestBundle_DefaultRunnerResources_DoesNotOverrideExplicitResources(t *testing.T) {
	proc := New(&dummyInspector{}).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessNestedSteps)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Image: "custom-image:latest",
								Resources: &testworkflowsv1.Resources{
									Requests: map[corev1.ResourceName]intstr.IntOrString{
										corev1.ResourceCPU:    intstr.FromString("200m"),
										corev1.ResourceMemory: intstr.FromString("256Mi"),
									},
								},
							},
						},
					},
				},
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
				},
			},
		},
	})

	require.NoError(t, err)
	allContainers := append(bundle.Job.Spec.Template.Spec.InitContainers, bundle.Job.Spec.Template.Spec.Containers...)
	// Find the container using the custom image; its explicitly set resources must not be replaced by defaults.
	found := false
	for _, c := range allContainers {
		if c.Image == "custom-image:latest" {
			found = true
			assert.Equalf(t, resource.MustParse("200m"), c.Resources.Requests[corev1.ResourceCPU], "container %s should keep explicit CPU request", c.Name)
			assert.Equalf(t, resource.MustParse("256Mi"), c.Resources.Requests[corev1.ResourceMemory], "container %s should keep explicit memory request", c.Name)
		}
	}
	require.True(t, found, "expected to find container with custom-image:latest")
}
