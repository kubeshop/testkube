package presets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

const (
	dummyUserId  = 1234
	dummyGroupId = 4321
)

var (
	dummyEntrypoint = []string{"/dummy-entrypoint", "entrypoint-arg"}
	dummyCmd        = []string{"/dummy-cmd", "cmd-arg"}
	dummyTime       = time.Date(2020, 10, 14, 1, 2, 3, 400, time.UTC)
)

type dummyInspector struct{}

func (*dummyInspector) Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*imageinspector.Info, error) {
	return &imageinspector.Info{
		Entrypoint: dummyEntrypoint,
		Cmd:        dummyCmd,
		User:       dummyUserId,
		Group:      dummyGroupId,
	}, nil
}

func (*dummyInspector) ResolveName(registry, image string) string {
	return image
}

var (
	ins        = &dummyInspector{}
	proc       = NewPro(ins)
	testConfig = testworkflowconfig.InternalConfig{
		Resource: testworkflowconfig.ResourceConfig{
			Id:     "dummy-id-abc",
			RootId: "dummy-id",
		},
	}
	envActions = actiontypes.EnvVarFrom(constants2.EnvGroupActions, false, false, constants2.EnvActions, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.SpecAnnotationFieldPath},
	})
	envInternal = actiontypes.EnvVarFrom(constants2.EnvGroupInternal, false, false, constants2.EnvInternalConfig, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.InternalAnnotationFieldPath},
	})
	envSignature = actiontypes.EnvVarFrom(constants2.EnvGroupInternal, false, false, constants2.EnvSignature, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.SignatureAnnotationFieldPath},
	})
	envDebugNode = actiontypes.EnvVarFrom(constants2.EnvGroupDebug, false, false, constants2.EnvNodeName, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
	})
	envDebugPod = actiontypes.EnvVarFrom(constants2.EnvGroupDebug, false, false, constants2.EnvPodName, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
	})
	envDebugNamespace = actiontypes.EnvVarFrom(constants2.EnvGroupDebug, false, false, constants2.EnvNamespaceName, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
	})
	envDebugServiceAccount = actiontypes.EnvVarFrom(constants2.EnvGroupDebug, false, false, constants2.EnvServiceAccountName, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
	})
)

func newResourceEnvVars(container string) []corev1.EnvVar {
	return []corev1.EnvVar{
		newResourceFieldRefEnvVar(constants2.EnvResourceRequestsCPU, container, "requests.cpu", resource.MustParse("1m")),
		newResourceFieldRefEnvVar(constants2.EnvResourceLimitsCPU, container, "limits.cpu", resource.MustParse("1m")),
		newResourceFieldRefEnvVar(constants2.EnvResourceRequestsMemory, container, "requests.memory", resource.Quantity{}),
		newResourceFieldRefEnvVar(constants2.EnvResourceLimitsMemory, container, "limits.memory", resource.Quantity{}),
	}
}

func newRuntimeEnvVar(container string) corev1.EnvVar {
	return actiontypes.EnvVar(constants2.EnvGroupRuntime, false, false, constants2.EnvContainerName, container)
}

func newResourceFieldRefEnvVar(envvar, container, resource string, divisor resource.Quantity) corev1.EnvVar {
	return actiontypes.EnvVarFrom(constants2.EnvGroupResources, false, false, envvar, corev1.EnvVarSource{
		ResourceFieldRef: &corev1.ResourceFieldSelector{ContainerName: container, Resource: resource, Divisor: divisor},
	})
}

func env(index int, computed bool, name, value string) corev1.EnvVar {
	return actiontypes.EnvVar(fmt.Sprintf("%d", index), computed, false, name, value)
}

func cmd(values ...string) *[]string {
	return &values
}

func cmdShell(shell string) *[]string {
	args := []string{"-c", "set -e\n" + shell}
	return &args
}

func and(values ...string) string {
	return strings.Join(values, "&&")
}

func getSpec(actions actiontypes.ActionGroups) string {
	v, err := json.Marshal(actions)
	if err != nil {
		panic(err)
	}
	return string(v)
}

func TestProcessEmpty(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has nothing to run")
}

func TestProcessBasic(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	assert.NoError(t, err)

	sig := res.Signature
	sigSerialized, _ := json.Marshal(sig)

	internalConfigSerialized, _ := json.Marshal(testConfig)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.Containers[0].VolumeMounts

	wantActions := actiontypes.NewActionGroups().
		Append(func(list actiontypes.ActionList) actiontypes.ActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// Joined as default image is used
				MutateContainer(sig[0].Ref(), testworkflowsv1.ContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantEnv := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv = append(append(wantEnv, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	want := batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dummy-id-abc",
			Labels: map[string]string{
				constants.ResourceIdLabelName:     "dummy-id-abc",
				constants.RootResourceIdLabelName: "dummy-id",
			},
			Annotations: nil,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.ResourceIdLabelName:     "dummy-id-abc",
						constants.RootResourceIdLabelName: "dummy-id",
					},
					Annotations: map[string]string{
						constants.SignatureAnnotationName:   string(sigSerialized),
						constants.InternalAnnotationName:    string(internalConfigSerialized),
						constants.SpecAnnotationName:        getSpec(wantActions),
						constants.ScheduledAtAnnotationName: dummyTime.Format(time.RFC3339Nano),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					EnableServiceLinks: common.Ptr(false),
					Volumes:            volumes,
					InitContainers:     []corev1.Container{},
					Containers: []corev1.Container{
						{
							Name:            "1",
							Image:           constants.DefaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/init", "0"},
							Env:             wantEnv,
							VolumeMounts:    volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsGroup: common.Ptr(constants.DefaultFsGroup),
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: common.Ptr(constants.DefaultFsGroup),
					},
				},
			},
		},
	}

	assert.Equal(t, want, res.Job)

	assert.Equal(t, 3, len(volumeMounts))
	assert.Equal(t, 3, len(volumes))
	assert.Equal(t, constants.DefaultInternalPath, volumeMounts[0].MountPath)
	assert.Equal(t, constants.DefaultTmpDirPath, volumeMounts[1].MountPath)
	assert.Equal(t, constants.DefaultDataPath, volumeMounts[2].MountPath)
	assert.True(t, volumeMounts[0].Name == volumes[0].Name)
	assert.True(t, volumeMounts[1].Name == volumes[1].Name)
	assert.True(t, volumeMounts[2].Name == volumes[2].Name)
}

func TestProcessShellWithNonStandardImage(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepDefaults:   testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{Image: "custom:1.2.3"}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
	assert.NoError(t, err)

	sig := res.Signature
	sigSerialized, _ := json.Marshal(sig)

	internalConfigSerialized, _ := json.Marshal(testConfig)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := actiontypes.NewActionGroups().
		Append(func(list actiontypes.ActionList) actiontypes.ActionList {
			return list.
				Setup(true, false, true).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list actiontypes.ActionList) actiontypes.ActionList {
			return list.
				MutateContainer(sig[0].Ref(), testworkflowsv1.ContainerConfig{
					Command: cmd("/.tktw/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantEnv1 := []corev1.EnvVar{
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv1 = append(append(wantEnv1, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantEnv2 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv2 = append(append(wantEnv2, newResourceEnvVars("2")...), newRuntimeEnvVar("2"))
	want := batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dummy-id-abc",
			Labels: map[string]string{
				constants.ResourceIdLabelName:     "dummy-id-abc",
				constants.RootResourceIdLabelName: "dummy-id",
			},
			Annotations: nil,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.ResourceIdLabelName:     "dummy-id-abc",
						constants.RootResourceIdLabelName: "dummy-id",
					},
					Annotations: map[string]string{
						constants.SignatureAnnotationName:   string(sigSerialized),
						constants.InternalAnnotationName:    string(internalConfigSerialized),
						constants.SpecAnnotationName:        getSpec(wantActions),
						constants.ScheduledAtAnnotationName: dummyTime.Format(time.RFC3339Nano),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					EnableServiceLinks: common.Ptr(false),
					Volumes:            volumes,
					InitContainers: []corev1.Container{
						{
							Name:            "1",
							Image:           constants.DefaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/init", "0"},
							Env:             wantEnv1,
							VolumeMounts:    volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsGroup: common.Ptr(int64(dummyGroupId)),
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "2",
							Image:           "custom:1.2.3",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/.tktw/init", "1"},
							Env:             wantEnv2,
							VolumeMounts:    volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsGroup: common.Ptr(int64(dummyGroupId)),
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: common.Ptr(int64(dummyGroupId)),
					},
				},
			},
		},
	}

	assert.Equal(t, want, res.Job)

	assert.Equal(t, 3, len(volumeMounts))
	assert.Equal(t, 3, len(volumes))
	assert.Equal(t, constants.DefaultInternalPath, volumeMounts[0].MountPath)
	assert.Equal(t, constants.DefaultTmpDirPath, volumeMounts[1].MountPath)
	assert.Equal(t, constants.DefaultDataPath, volumeMounts[2].MountPath)
	assert.True(t, volumeMounts[0].Name == volumes[0].Name)
	assert.True(t, volumeMounts[1].Name == volumes[1].Name)
	assert.True(t, volumeMounts[2].Name == volumes[2].Name)
}

func TestProcessBasicEnvReference(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepDefaults: testworkflowsv1.StepDefaults{
					Container: &testworkflowsv1.ContainerConfig{
						Env: []testworkflowsv1.EnvVar{
							{EnvVar: corev1.EnvVar{Name: "ZERO", Value: "foo"}},
							{EnvVar: corev1.EnvVar{Name: "UNDETERMINED", Value: "{{call(abc)}}xxx"}},
							{EnvVar: corev1.EnvVar{Name: "INPUT", Value: "{{env.ZERO}}bar"}},
							{EnvVar: corev1.EnvVar{Name: "NEXT", Value: "foo{{env.UNDETERMINED}}{{env.LAST}}"}},
							{EnvVar: corev1.EnvVar{Name: "LAST", Value: "foo{{env.INPUT}}bar"}},
						},
					},
				}, StepOperations: testworkflowsv1.StepOperations{
					Shell: "shell-test",
				}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.Containers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantEnv := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		env(0, false, "ZERO", "foo"),
		env(0, true, "UNDETERMINED", "{{call(abc)}}xxx"),
		env(0, false, "INPUT", "foobar"),
		env(0, true, "NEXT", "foo{{env.UNDETERMINED}}foofoobarbar"),
		env(0, false, "LAST", "foofoobarbar"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv = append(append(wantEnv, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantPod := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers:     []corev1.Container{},
		Containers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env:             wantEnv,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(constants.DefaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, wantPod, res.Job.Spec.Template.Spec)
	assert.Equal(t, wantActions, res.LiteActions())
}

func TestProcessMultipleSteps(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test"}},
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// Joined as default container is used
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test-2"),
				}).
				Start(sig[1].Ref()).
				Execute(sig[1].Ref(), false, false).
				End(sig[1].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantEnv1 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv1 = append(append(wantEnv1, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantEnv2 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv2 = append(append(wantEnv2, newResourceEnvVars("2")...), newRuntimeEnvVar("2"))
	want := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env:             wantEnv1,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env:             wantEnv2,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(constants.DefaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
	assert.Equal(t, wantActions, res.LiteActions())
}

func TestProcessNestedSteps(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "A"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test"}},
				{
					StepMeta: testworkflowsv1.StepMeta{Name: "B"},
					Steps: []testworkflowsv1.Step{
						{StepMeta: testworkflowsv1.StepMeta{Name: "C"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-2"}},
						{StepMeta: testworkflowsv1.StepMeta{Name: "D"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-3"}},
					},
				},
				{StepMeta: testworkflowsv1.StepMeta{Name: "E"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-4"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Declare(sig[1].Children()[0].Ref(), sig[0].Ref(), constants.RootOperationName, sig[1].Ref()).
				Declare(sig[1].Children()[1].Ref(), and(sig[1].Children()[0].Ref(), sig[0].Ref()), constants.RootOperationName, sig[1].Ref()).
				Declare(sig[2].Ref(), and(sig[1].Ref(), sig[0].Ref()), constants.RootOperationName).
				Result(sig[1].Ref(), and(sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref())).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref(), sig[2].Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// Joined as default container is used
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
				Start(sig[1].Ref()).
				CurrentStatus(and(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test-2"),
				}).
				Start(sig[1].Children()[0].Ref()).
				Execute(sig[1].Children()[0].Ref(), false, false).
				End(sig[1].Children()[0].Ref()).
				CurrentStatus(and(sig[1].Children()[0].Ref(), sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test-3"),
				}).
				Start(sig[1].Children()[1].Ref()).
				Execute(sig[1].Children()[1].Ref(), false, false).
				End(sig[1].Children()[1].Ref()).
				End(sig[1].Ref()).
				CurrentStatus(and(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test-4"),
				}).
				Start(sig[2].Ref()).
				Execute(sig[2].Ref(), false, false).
				End(sig[2].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantEnv1 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv1 = append(append(wantEnv1, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantEnv2 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv2 = append(append(wantEnv2, newResourceEnvVars("2")...), newRuntimeEnvVar("2"))
	wantEnv3 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv3 = append(append(wantEnv3, newResourceEnvVars("3")...), newRuntimeEnvVar("3"))
	wantEnv4 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv4 = append(append(wantEnv4, newResourceEnvVars("4")...), newRuntimeEnvVar("4"))
	want := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env:             wantEnv1,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env:             wantEnv2,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "3",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "2"},
				Env:             wantEnv3,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "4",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "3"},
				Env:             wantEnv4,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(constants.DefaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, wantActions, res.LiteActions())
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessLocalContent(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{
					Shell: "shell-test",
				}, StepSource: testworkflowsv1.StepSource{
					Content: &testworkflowsv1.Content{
						Files: []testworkflowsv1.ContentFile{{
							Path:    "/some/path",
							Content: `some-{{"{{"}}content`,
						}},
					},
				}},
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	assert.NoError(t, err)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.Containers[0].VolumeMounts
	volumeMountsWithContent := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantEnv1 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv1 = append(append(wantEnv1, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantEnv2 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv2 = append(append(wantEnv2, newResourceEnvVars("2")...), newRuntimeEnvVar("2"))
	want := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env:             wantEnv1,
				VolumeMounts:    volumeMountsWithContent,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env:             wantEnv2,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(constants.DefaultFsGroup),
		},
	}

	assert.Equal(t, want, res.Job.Spec.Template.Spec)
	assert.Equal(t, 3, len(volumeMounts))
	assert.Equal(t, 4, len(volumeMountsWithContent))
	assert.Equal(t, volumeMounts, volumeMountsWithContent[:3])
	assert.Equal(t, "/some/path", volumeMountsWithContent[3].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMountsWithContent[3].Name, volumes[3].Name)
	assert.Equal(t, volumes[3].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMountsWithContent[3].SubPath])
}

func TestProcessGlobalContent(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: &testworkflowsv1.Content{
					Files: []testworkflowsv1.ContentFile{{
						Path:    "/some/path",
						Content: `some-{{"{{"}}content`,
					}},
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test"}},
				{StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	assert.NoError(t, err)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantEnv1 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
		envDebugNode,
		envDebugPod,
		envDebugNamespace,
		envDebugServiceAccount,
		envActions,
		envInternal,
		envSignature,
	}
	wantEnv1 = append(append(wantEnv1, newResourceEnvVars("1")...), newRuntimeEnvVar("1"))
	wantEnv2 := []corev1.EnvVar{
		env(0, false, "CI", "1"),
	}
	wantEnv2 = append(append(wantEnv2, newResourceEnvVars("2")...), newRuntimeEnvVar("2"))
	want := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env:             wantEnv1,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env:             wantEnv2,
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(constants.DefaultFsGroup),
		},
	}

	v, _ := json.Marshal(want)
	fmt.Println(string(v))

	assert.Equal(t, want, res.Job.Spec.Template.Spec)
	assert.Equal(t, 4, len(volumeMounts))
	assert.Equal(t, "/some/path", volumeMounts[3].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMounts[3].Name, volumes[3].Name)
	assert.Equal(t, volumes[3].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMounts[3].SubPath])
}

func TestProcessEscapedAnnotations(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					Annotations: map[string]string{
						"vault.hashicorp.com/agent-inject-template-database-config.txt": `{{"{{"}}- with secret "internal/data/database/config" -}}{{"{{"}} .Data.data.username }}@{{"{{"}} .Data.data.password }}{{"{{"}}- end -}}`,
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	assert.NoError(t, err)
	assert.Equal(t, `{{- with secret "internal/data/database/config" -}}{{ .Data.data.username }}@{{ .Data.data.password }}{{- end -}}`, res.Job.Spec.Template.Annotations["vault.hashicorp.com/agent-inject-template-database-config.txt"])
}

func TestProcessShell(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// Joined together as default image is used
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcessConsecutiveAlways(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{
					IsolatedContainers: common.Ptr(true),
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
				{StepMeta: testworkflowsv1.StepMeta{Condition: "always"}, StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[0].Ref()).
			Execute(sig[0].Ref(), false, false).
			End(sig[0].Ref()).
			CurrentStatus(and(sig[0].Ref(), constants.RootOperationName))
	}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[1].Ref()).
			Execute(sig[1].Ref(), false, false).
			End(sig[1].Ref()).
			End(constants.RootOperationName).
			End("")
	})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcessNestedCondition(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{
					IsolatedContainers: common.Ptr(true),
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
				{StepMeta: testworkflowsv1.StepMeta{Condition: "always"}, Steps: []testworkflowsv1.Step{
					{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
				}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[0].Ref()).
			Execute(sig[0].Ref(), false, false).
			End(sig[0].Ref()).
			CurrentStatus(and(sig[0].Ref(), constants.RootOperationName))
	}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[1].Ref()).
			Execute(sig[1].Ref(), false, false).
			End(sig[1].Ref()).
			End(constants.RootOperationName).
			End("")
	})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcessConditionWithMultipleOperations(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{
					IsolatedContainers: common.Ptr(true),
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")}}},
				{StepMeta: testworkflowsv1.StepMeta{Condition: "always"}, StepOperations: testworkflowsv1.StepOperations{
					Run:   &testworkflowsv1.StepRun{Shell: common.Ptr("shell-test")},
					Shell: "shell-test-2",
				}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature
	virtual := res.FullSignature[1]

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(virtual.Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), "true", constants.RootOperationName, virtual.Ref()).
				Declare(sig[2].Ref(), "true", constants.RootOperationName, virtual.Ref()).
				Result(virtual.Ref(), and(sig[1].Ref(), sig[2].Ref())).
				Result(constants.RootOperationName, and(sig[0].Ref(), virtual.Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[0].Ref()).
			Execute(sig[0].Ref(), false, false).
			End(sig[0].Ref()).
			CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
			Start(virtual.Ref()).
			CurrentStatus(and(virtual.Ref(), sig[0].Ref(), constants.RootOperationName))
	}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test"),
			}).
			Start(sig[1].Ref()).
			Execute(sig[1].Ref(), false, false).
			End(sig[1].Ref()).
			CurrentStatus(and(sig[1].Ref(), virtual.Ref(), sig[0].Ref(), constants.RootOperationName))

	}).Append(func(list lite.LiteActionList) lite.LiteActionList {
		return list.
			MutateContainer(lite.LiteContainerConfig{
				Command: cmd("/.tktw-bin/sh"),
				Args:    cmdShell("shell-test-2"),
			}).
			Start(sig[2].Ref()).
			Execute(sig[2].Ref(), false, false).
			End(sig[2].Ref()).
			End(virtual.Ref()).
			End(constants.RootOperationName).
			End("")
	})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcessNamedGroupWithSkippedSteps(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{
					IsolatedContainers: common.Ptr(true),
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "test-group", Condition: "always"}, Steps: []testworkflowsv1.Step{
					{StepMeta: testworkflowsv1.StepMeta{Condition: "never"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-1"}},
					{StepMeta: testworkflowsv1.StepMeta{Condition: "never"}, StepOperations: testworkflowsv1.StepOperations{Shell: "shell-test-2"}},
				}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// configure
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[0].Children()[0].Ref(), "false").
				Declare(sig[0].Children()[1].Ref(), "false").
				Result(sig[0].Ref(), "true").
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// start the group
				Start(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).

				// void operations
				Start(sig[0].Children()[0].Ref()).
				End(sig[0].Children()[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
				Start(sig[0].Children()[1].Ref()).
				End(sig[0].Children()[1].Ref()).

				// finish all
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcess_ConditionAlways(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "test-command-1"}},
				{
					StepMeta: testworkflowsv1.StepMeta{Condition: "always"},
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Env: []testworkflowsv1.EnvVar{
									{EnvVar: corev1.EnvVar{Name: "result", Value: "{{passed}}"}},
								},
							},
							Shell: common.Ptr("echo $result"),
						},
					},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// configure
				Setup(false, false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).

				// initialize
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName).

				// start first container
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("test-command-1"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, false).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw-bin/sh"),
					Args:    cmdShell("echo $result"),
				}).
				Start(sig[1].Ref()).
				Execute(sig[1].Ref(), false, false).
				End(sig[1].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcess_PureShellAtTheEnd(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepMeta: testworkflowsv1.StepMeta{Pure: common.Ptr(true)},
					StepDefaults: testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{
						Image: "custom-image:1.2.3",
					}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "test-command-1"},
				},
				{
					StepMeta:       testworkflowsv1.StepMeta{Pure: common.Ptr(true)},
					StepOperations: testworkflowsv1.StepOperations{Shell: "test-command-2"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// configure
				Setup(true, false, true).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).

				// initialize
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// start first container
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw/bin/sh"),
					Args:    cmdShell("test-command-1"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, true).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw/bin/sh"),
					Args:    cmdShell("test-command-2"),
				}).
				Start(sig[1].Ref()).
				Execute(sig[1].Ref(), false, true).
				End(sig[1].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}

func TestProcess_MergingActions(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{Delay: "1s"},
				},
				{
					StepDefaults: testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{
						Image: "custom-image:1.2.3",
					}},
					StepOperations: testworkflowsv1.StepOperations{Shell: "test-command"},
				},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
	sig := res.Signature

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// configure
				Setup(true, false, true).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).

				// initialize
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				// start first container
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("sleep"),
					Args:    cmd("1"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false, true).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/.tktw/bin/sh"),
					Args:    cmdShell("test-command"),
				}).
				Start(sig[1].Ref()).
				Execute(sig[1].Ref(), false, false).
				End(sig[1].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, wantActions, res.LiteActions())
	assert.Equal(t, res.Job.Spec.Template.Spec.Containers[0].Image, "custom-image:1.2.3")
}
