package presets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
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
	ins         = &dummyInspector{}
	proc        = NewPro(ins)
	execMachine = expressions.NewMachine().
			Register("resource.root", "dummy-id").
			Register("resource.id", "dummy-id-abc")
	envActions = actiontypes.EnvVarFrom(constants2.EnvGroupActions, false, false, constants2.EnvActions, corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.SpecAnnotationFieldPath},
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

	_, err := proc.Bundle(context.Background(), wf, execMachine)

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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	sig := res.Signature
	sigSerialized, _ := json.Marshal(sig)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := actiontypes.NewActionGroups().
		Append(func(list actiontypes.ActionList) actiontypes.ActionList {
			return list.
				Setup(false, false).
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
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	want := batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dummy-id-abc",
			Labels: map[string]string{
				constants.ResourceIdLabelName:     "dummy-id-abc",
				constants.RootResourceIdLabelName: "dummy-id",
			},
			Annotations: map[string]string{
				constants.SignatureAnnotationName: string(sigSerialized),
			},
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
						constants.SpecAnnotationName: getSpec(wantActions),
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
							Env: []corev1.EnvVar{
								envDebugNode,
								envDebugPod,
								envDebugNamespace,
								envDebugServiceAccount,
								envActions,
							},
							VolumeMounts: volumeMounts,
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
							Env: []corev1.EnvVar{
								env(0, false, "CI", "1"),
							},
							VolumeMounts: volumeMounts,
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

	assert.Equal(t, 2, len(volumeMounts))
	assert.Equal(t, 2, len(volumes))
	assert.Equal(t, constants.DefaultInternalPath, volumeMounts[0].MountPath)
	assert.Equal(t, constants.DefaultDataPath, volumeMounts[1].MountPath)
	assert.True(t, volumeMounts[0].Name == volumes[0].Name)
	assert.True(t, volumeMounts[1].Name == volumes[1].Name)
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	sig := res.Signature
	sigSerialized, _ := json.Marshal(sig)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := actiontypes.NewActionGroups().
		Append(func(list actiontypes.ActionList) actiontypes.ActionList {
			return list.
				Setup(true, true).
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
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	want := batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dummy-id-abc",
			Labels: map[string]string{
				constants.ResourceIdLabelName:     "dummy-id-abc",
				constants.RootResourceIdLabelName: "dummy-id",
			},
			Annotations: map[string]string{
				constants.SignatureAnnotationName: string(sigSerialized),
			},
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
						constants.SpecAnnotationName: getSpec(wantActions),
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
							Env: []corev1.EnvVar{
								envDebugNode,
								envDebugPod,
								envDebugNamespace,
								envDebugServiceAccount,
								envActions,
							},
							VolumeMounts: volumeMounts,
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
							Env: []corev1.EnvVar{
								env(0, false, "CI", "1"),
							},
							VolumeMounts: volumeMounts,
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

	assert.Equal(t, 2, len(volumeMounts))
	assert.Equal(t, 2, len(volumes))
	assert.Equal(t, constants.DefaultInternalPath, volumeMounts[0].MountPath)
	assert.Equal(t, constants.DefaultDataPath, volumeMounts[1].MountPath)
	assert.True(t, volumeMounts[0].Name == volumes[0].Name)
	assert.True(t, volumeMounts[1].Name == volumes[1].Name)
}

func TestProcessBasicEnvReference(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepDefaults: testworkflowsv1.StepDefaults{
					Container: &testworkflowsv1.ContainerConfig{
						Env: []corev1.EnvVar{
							{Name: "ZERO", Value: "foo"},
							{Name: "UNDETERMINED", Value: "{{call(abc)}}xxx"},
							{Name: "INPUT", Value: "{{env.ZERO}}bar"},
							{Name: "NEXT", Value: "foo{{env.UNDETERMINED}}{{env.LAST}}"},
							{Name: "LAST", Value: "foo{{env.INPUT}}bar"},
						},
					},
				}, StepOperations: testworkflowsv1.StepOperations{
					Shell: "shell-test",
				}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	wantPod := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		EnableServiceLinks: common.Ptr(false),
		Volumes:            volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "1",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "0"},
				Env: []corev1.EnvVar{
					envDebugNode,
					envDebugPod,
					envDebugNamespace,
					envDebugServiceAccount,
					envActions,
				},
				VolumeMounts: volumeMounts,
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
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
					env(0, false, "ZERO", "foo"),
					env(0, true, "UNDETERMINED", "{{call(abc)}}xxx"),
					env(0, false, "INPUT", "foobar"),
					env(0, true, "NEXT", "foo{{env.UNDETERMINED}}foofoobarbar"),
					env(0, false, "LAST", "foofoobarbar"),
				},
				VolumeMounts: volumeMounts,
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Declare(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName).
				Result(constants.RootOperationName, and(sig[0].Ref(), sig[1].Ref())).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test-2"),
				}).
				Start(sig[1].Ref()).
				Execute(sig[1].Ref(), false).
				End(sig[1].Ref()).
				End(constants.RootOperationName).
				End("")
		})

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
				Env: []corev1.EnvVar{
					envDebugNode,
					envDebugPod,
					envDebugNamespace,
					envDebugServiceAccount,
					envActions,
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "3",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "2"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	wantActions := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false).
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
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				CurrentStatus(and(sig[0].Ref(), constants.RootOperationName)).
				Start(sig[1].Ref()).
				CurrentStatus(and(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test-2"),
				}).
				Start(sig[1].Children()[0].Ref()).
				Execute(sig[1].Children()[0].Ref(), false).
				End(sig[1].Children()[0].Ref()).
				CurrentStatus(and(sig[1].Children()[0].Ref(), sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test-3"),
				}).
				Start(sig[1].Children()[1].Ref()).
				Execute(sig[1].Children()[1].Ref(), false).
				End(sig[1].Children()[1].Ref()).
				End(sig[1].Ref()).
				CurrentStatus(and(sig[1].Ref(), sig[0].Ref(), constants.RootOperationName))
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test-4"),
				}).
				Start(sig[2].Ref()).
				Execute(sig[2].Ref(), false).
				End(sig[2].Ref()).
				End(constants.RootOperationName).
				End("")
		})

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
				Env: []corev1.EnvVar{
					envDebugNode,
					envDebugPod,
					envDebugNamespace,
					envDebugServiceAccount,
					envActions,
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "3",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "2"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "4",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "3"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "5",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "4"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts
	volumeMountsWithContent := res.Job.Spec.Template.Spec.InitContainers[1].VolumeMounts

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
				Env: []corev1.EnvVar{
					envDebugNode,
					envDebugPod,
					envDebugNamespace,
					envDebugServiceAccount,
					envActions,
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMountsWithContent,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "3",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "2"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
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
	assert.Equal(t, 2, len(volumeMounts))
	assert.Equal(t, 3, len(volumeMountsWithContent))
	assert.Equal(t, volumeMounts, volumeMountsWithContent[:2])
	assert.Equal(t, "/some/path", volumeMountsWithContent[2].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMountsWithContent[2].Name, volumes[2].Name)
	assert.Equal(t, volumes[2].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMountsWithContent[2].SubPath])
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

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
				Env: []corev1.EnvVar{
					envDebugNode,
					envDebugPod,
					envDebugNamespace,
					envDebugServiceAccount,
					envActions,
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
			{
				Name:            "2",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "1"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(constants.DefaultFsGroup),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "3",
				Image:           constants.DefaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/init", "2"},
				Env: []corev1.EnvVar{
					env(0, false, "CI", "1"),
				},
				VolumeMounts: volumeMounts,
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
	assert.Equal(t, "/some/path", volumeMounts[2].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMounts[2].Name, volumes[2].Name)
	assert.Equal(t, volumes[2].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMounts[2].SubPath])
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	want := lite.NewLiteActionGroups().
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				Setup(false, false).
				Declare(constants.RootOperationName, "true").
				Declare(sig[0].Ref(), "true", constants.RootOperationName).
				Result(constants.RootOperationName, sig[0].Ref()).
				Result("", constants.RootOperationName).
				Start("").
				CurrentStatus("true").
				Start(constants.RootOperationName).
				CurrentStatus(constants.RootOperationName)
		}).
		Append(func(list lite.LiteActionList) lite.LiteActionList {
			return list.
				MutateContainer(lite.LiteContainerConfig{
					Command: cmd("/bin/sh"),
					Args:    cmdShell("shell-test"),
				}).
				Start(sig[0].Ref()).
				Execute(sig[0].Ref(), false).
				End(sig[0].Ref()).
				End(constants.RootOperationName).
				End("")
		})

	assert.NoError(t, err)
	assert.Equal(t, want, res.LiteActions())
}
