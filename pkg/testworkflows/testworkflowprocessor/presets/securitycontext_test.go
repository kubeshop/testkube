package presets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

// TestSecurityContextPropagation_SpecLevel verifies that spec.container.securityContext
// is propagated to all init and main containers.
func TestSecurityContextPropagation_SpecLevel(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:             common.Ptr(true),
						RunAsUser:                common.Ptr(int64(1000)),
						ReadOnlyRootFilesystem:   common.Ptr(true),
						AllowPrivilegeEscalation: common.Ptr(false),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "custom:1.2.3"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "step-1"},
				},
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "custom:4.5.6"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "step-2"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	// All containers should have the spec-level securityContext fields propagated
	for _, c := range allContainers {
		require.NotNilf(t, c.SecurityContext, "container %s: SecurityContext should not be nil", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.RunAsNonRoot, "container %s: RunAsNonRoot", c.Name)
		assert.Equalf(t, common.Ptr(int64(1000)), c.SecurityContext.RunAsUser, "container %s: RunAsUser", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.ReadOnlyRootFilesystem, "container %s: ReadOnlyRootFilesystem", c.Name)
		assert.Equalf(t, common.Ptr(false), c.SecurityContext.AllowPrivilegeEscalation, "container %s: AllowPrivilegeEscalation", c.Name)
	}
}

// TestSecurityContextPropagation_SpecLevelDefaultImage verifies that spec.container.securityContext
// is propagated to containers using the default init image (no custom image).
func TestSecurityContextPropagation_SpecLevelDefaultImage(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot: common.Ptr(true),
						RunAsUser:    common.Ptr(int64(2000)),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}},
				{StepOperations: testworkflowsv1.StepOperations{Shell: "echo world"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	for _, c := range allContainers {
		require.NotNilf(t, c.SecurityContext, "container %s: SecurityContext should not be nil", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.RunAsNonRoot, "container %s: RunAsNonRoot", c.Name)
		assert.Equalf(t, common.Ptr(int64(2000)), c.SecurityContext.RunAsUser, "container %s: RunAsUser", c.Name)
	}
}

// TestSecurityContextPropagation_StepLevel verifies that step.container.securityContext
// is applied to the step's own containers and propagated to nested sub-steps.
func TestSecurityContextPropagation_StepLevel(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults: testworkflowsv1.StepDefaults{
						Container: &testworkflowsv1.ContainerConfig{
							Image: "custom:1.0",
							SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
								RunAsNonRoot: common.Ptr(true),
								RunAsUser:    common.Ptr(int64(3000)),
							}),
						},
					},
					StepOperations: testworkflowsv1.StepOperations{Shell: "step-with-sc"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec

	// The main container (or the container running the step) should have the SecurityContext
	// Find the container with the custom image
	var mainContainer *corev1.Container
	allContainers := append(spec.InitContainers, spec.Containers...)
	for i := range allContainers {
		if allContainers[i].Image == "custom:1.0" {
			mainContainer = &allContainers[i]
			break
		}
	}

	require.NotNil(t, mainContainer, "should find container with custom image")
	require.NotNil(t, mainContainer.SecurityContext, "SecurityContext should not be nil")
	assert.Equal(t, common.Ptr(true), mainContainer.SecurityContext.RunAsNonRoot)
	assert.Equal(t, common.Ptr(int64(3000)), mainContainer.SecurityContext.RunAsUser)
}

// TestSecurityContextPropagation_StepOverridesSpec verifies that step-level
// securityContext merges with and overrides spec-level securityContext.
func TestSecurityContextPropagation_StepOverridesSpec(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:           common.Ptr(true),
						RunAsUser:              common.Ptr(int64(1000)),
						ReadOnlyRootFilesystem: common.Ptr(true),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults: testworkflowsv1.StepDefaults{
						Container: &testworkflowsv1.ContainerConfig{
							Image: "custom:override",
							SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
								// Override RunAsUser, keep RunAsNonRoot and ReadOnlyRootFilesystem from spec
								RunAsUser: common.Ptr(int64(2000)),
							}),
						},
					},
					StepOperations: testworkflowsv1.StepOperations{Shell: "step-override"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	// Find the container with the custom image
	var mainContainer *corev1.Container
	for i := range allContainers {
		if allContainers[i].Image == "custom:override" {
			mainContainer = &allContainers[i]
			break
		}
	}

	require.NotNil(t, mainContainer, "should find container with custom image")
	require.NotNil(t, mainContainer.SecurityContext, "SecurityContext should not be nil")
	// Step overrides RunAsUser
	assert.Equal(t, common.Ptr(int64(2000)), mainContainer.SecurityContext.RunAsUser)
	// Spec-level values are inherited
	assert.Equal(t, common.Ptr(true), mainContainer.SecurityContext.RunAsNonRoot)
	assert.Equal(t, common.Ptr(true), mainContainer.SecurityContext.ReadOnlyRootFilesystem)
}

// TestSecurityContextPropagation_NestedSteps verifies that securityContext
// propagates through nested step hierarchies.
func TestSecurityContextPropagation_NestedSteps(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot: common.Ptr(true),
						RunAsUser:    common.Ptr(int64(1000)),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepMeta: testworkflowsv1.StepMeta{Name: "parent"},
					Steps: []testworkflowsv1.Step{
						{
							StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "nested:1.0"}},
							StepOperations: testworkflowsv1.StepOperations{Shell: "nested-step"},
						},
					},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	// Find the container with the nested image
	var nestedContainer *corev1.Container
	for i := range allContainers {
		if allContainers[i].Image == "nested:1.0" {
			nestedContainer = &allContainers[i]
			break
		}
	}

	require.NotNil(t, nestedContainer, "should find container with nested image")
	require.NotNil(t, nestedContainer.SecurityContext, "SecurityContext should not be nil")
	assert.Equal(t, common.Ptr(true), nestedContainer.SecurityContext.RunAsNonRoot)
	assert.Equal(t, common.Ptr(int64(1000)), nestedContainer.SecurityContext.RunAsUser)
}

// TestSecurityContextPropagation_RunStep verifies that securityContext from spec.container
// propagates to step.run containers when the run step doesn't set its own securityContext.
func TestSecurityContextPropagation_RunStep(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:             common.Ptr(true),
						AllowPrivilegeEscalation: common.Ptr(false),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Image:   "runner:latest",
								Command: &[]string{"run-cmd"},
							},
						},
					},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	// Find the container with the runner image
	var runnerContainer *corev1.Container
	for i := range allContainers {
		if allContainers[i].Image == "runner:latest" {
			runnerContainer = &allContainers[i]
			break
		}
	}

	require.NotNil(t, runnerContainer, "should find container with runner image")
	require.NotNil(t, runnerContainer.SecurityContext, "SecurityContext should not be nil")
	assert.Equal(t, common.Ptr(true), runnerContainer.SecurityContext.RunAsNonRoot)
	assert.Equal(t, common.Ptr(false), runnerContainer.SecurityContext.AllowPrivilegeEscalation)
}

// TestSecurityContextPropagation_RunStepOverridesSpec verifies that a run step's
// securityContext merges with the spec-level securityContext.
func TestSecurityContextPropagation_RunStepOverridesSpec(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:           common.Ptr(true),
						ReadOnlyRootFilesystem: common.Ptr(true),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Image:   "runner:override",
								Command: &[]string{"run-cmd"},
								SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
									// Override ReadOnlyRootFilesystem, keep RunAsNonRoot from spec
									ReadOnlyRootFilesystem: common.Ptr(false),
								}),
							},
						},
					},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	// Find the container with the runner image
	var runnerContainer *corev1.Container
	for i := range allContainers {
		if allContainers[i].Image == "runner:override" {
			runnerContainer = &allContainers[i]
			break
		}
	}

	require.NotNil(t, runnerContainer, "should find container with runner image")
	require.NotNil(t, runnerContainer.SecurityContext, "SecurityContext should not be nil")
	// Spec-level inherited
	assert.Equal(t, common.Ptr(true), runnerContainer.SecurityContext.RunAsNonRoot)
	// Step-level overridden
	assert.Equal(t, common.Ptr(false), runnerContainer.SecurityContext.ReadOnlyRootFilesystem)
}

// TestSecurityContextPropagation_InitContainersInherit verifies that init containers
// (setup steps) properly inherit the spec-level securityContext even when
// the step uses a non-default image.
func TestSecurityContextPropagation_InitContainersInherit(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:             common.Ptr(true),
						AllowPrivilegeEscalation: common.Ptr(false),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "step1:v1"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "first-step"},
				},
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "step2:v2"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "second-step"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec

	// Verify init containers have the spec-level securityContext
	for _, c := range spec.InitContainers {
		require.NotNilf(t, c.SecurityContext, "init container %s: SecurityContext should not be nil", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.RunAsNonRoot, "init container %s: RunAsNonRoot", c.Name)
		assert.Equalf(t, common.Ptr(false), c.SecurityContext.AllowPrivilegeEscalation, "init container %s: AllowPrivilegeEscalation", c.Name)
	}

	// Verify main containers also have the spec-level securityContext
	for _, c := range spec.Containers {
		require.NotNilf(t, c.SecurityContext, "main container %s: SecurityContext should not be nil", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.RunAsNonRoot, "main container %s: RunAsNonRoot", c.Name)
		assert.Equalf(t, common.Ptr(false), c.SecurityContext.AllowPrivilegeEscalation, "main container %s: AllowPrivilegeEscalation", c.Name)
	}
}

// TestSecurityContextPropagation_Capabilities verifies that Capabilities in
// securityContext are properly propagated.
func TestSecurityContextPropagation_Capabilities(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add:  []corev1.Capability{"NET_ADMIN"},
							Drop: []corev1.Capability{"ALL"},
						},
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "custom:caps"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "test-caps"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	for _, c := range allContainers {
		require.NotNilf(t, c.SecurityContext, "container %s: SecurityContext should not be nil", c.Name)
		require.NotNilf(t, c.SecurityContext.Capabilities, "container %s: Capabilities should not be nil", c.Name)
		assert.Containsf(t, c.SecurityContext.Capabilities.Add, corev1.Capability("NET_ADMIN"), "container %s: should have NET_ADMIN", c.Name)
		assert.Containsf(t, c.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"), "container %s: should drop ALL", c.Name)
		assert.Lenf(t, c.SecurityContext.Capabilities.Drop, 1, "container %s: should have exactly one Drop entry", c.Name)
		assert.Lenf(t, c.SecurityContext.Capabilities.Add, 1, "container %s: should have exactly one Add entry", c.Name)
	}
}

// TestSecurityContextPropagation_PodSecurityContextCombined verifies that
// pod-level and container-level securityContext work together correctly.
func TestSecurityContextPropagation_PodSecurityContextCombined(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					SecurityContext: testworkflowsv1.WorkflowPodSecurityContextFromKube(&corev1.PodSecurityContext{
						RunAsNonRoot: common.Ptr(true),
						FSGroup:      common.Ptr(int64(2000)),
					}),
				},
				Container: &testworkflowsv1.ContainerConfig{
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsUser:                common.Ptr(int64(1000)),
						AllowPrivilegeEscalation: common.Ptr(false),
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "custom:combined"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "test-combined"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec

	// Pod-level SecurityContext
	require.NotNil(t, spec.SecurityContext)
	assert.Equal(t, common.Ptr(true), spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, common.Ptr(int64(2000)), spec.SecurityContext.FSGroup)

	// Container-level SecurityContext
	allContainers := append(spec.InitContainers, spec.Containers...)
	for _, c := range allContainers {
		require.NotNilf(t, c.SecurityContext, "container %s: SecurityContext should not be nil", c.Name)
		assert.Equalf(t, common.Ptr(int64(1000)), c.SecurityContext.RunAsUser, "container %s: RunAsUser", c.Name)
		assert.Equalf(t, common.Ptr(false), c.SecurityContext.AllowPrivilegeEscalation, "container %s: AllowPrivilegeEscalation", c.Name)
	}
}

// TestSecurityContextPropagation_PureRunStepCapabilitiesDrop verifies that for a workflow
// with spec.container.securityContext including capabilities.drop=["ALL"],
// all containers (init and main) have this setting propagated — including
// steps with pure: true and run.shell.
func TestSecurityContextPropagation_PureRunStepCapabilitiesDrop(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: &testworkflowsv1.Content{
					Git: &testworkflowsv1.ContentGit{
						Uri:      "https://github.com/example/example",
						Revision: "main",
						Paths:    []string{"tests/"},
					},
				},
				Container: &testworkflowsv1.ContainerConfig{
					Image:      "microsoft/playwright:v1.44.0-jammy",
					WorkingDir: common.Ptr("/data/repo"),
					SecurityContext: testworkflowsv1.WorkflowSecurityContextFromKube(&corev1.SecurityContext{
						RunAsNonRoot:             common.Ptr(true),
						AllowPrivilegeEscalation: common.Ptr(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					}),
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepMeta: testworkflowsv1.StepMeta{
						Name: "Install dependencies",
						Pure: common.Ptr(true),
					},
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							Shell: common.Ptr("npm ci"),
						},
					},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	require.NoError(t, err)

	spec := res.Job.Spec.Template.Spec
	allContainers := append(spec.InitContainers, spec.Containers...)

	for _, c := range allContainers {
		require.NotNilf(t, c.SecurityContext, "container %s: SecurityContext should not be nil", c.Name)
		require.NotNilf(t, c.SecurityContext.Capabilities, "container %s: Capabilities should not be nil", c.Name)
		assert.Containsf(t, c.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"),
			"container %s: should have capabilities.drop=[ALL]", c.Name)
		assert.Lenf(t, c.SecurityContext.Capabilities.Drop, 1,
			"container %s: should have exactly one Drop entry (no duplicates)", c.Name)
		assert.Equalf(t, common.Ptr(true), c.SecurityContext.RunAsNonRoot,
			"container %s: RunAsNonRoot", c.Name)
		assert.Equalf(t, common.Ptr(false), c.SecurityContext.AllowPrivilegeEscalation,
			"container %s: AllowPrivilegeEscalation", c.Name)
		require.NotNilf(t, c.SecurityContext.SeccompProfile,
			"container %s: SeccompProfile should not be nil", c.Name)
		assert.Equalf(t, corev1.SeccompProfileTypeRuntimeDefault, c.SecurityContext.SeccompProfile.Type,
			"container %s: SeccompProfile.Type", c.Name)
	}
}
