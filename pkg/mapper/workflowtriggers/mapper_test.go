package workflowtriggers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapCRDToAPI_flattensSpecAndPreservesFields(t *testing.T) {
	delay := metav1.Duration{Duration: 30 * time.Second}
	status := workflowtriggersv1.WorkflowTriggerConditionStatusTrue
	crd := &workflowtriggersv1.WorkflowTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "canary",
			Namespace: "prod",
			Labels:    map[string]string{"app": "api"},
		},
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			Disabled: true,
			Watch: &workflowtriggersv1.WorkflowTriggerWatch{
				Resource: workflowtriggersv1.WorkflowTriggerResource{
					Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout",
				},
			},
			When: workflowtriggersv1.WorkflowTriggerWhen{
				Event: "modified",
				Git: &workflowtriggersv1.WorkflowTriggerWhenGitSpec{
					Uri:      "https://github.com/kubeshop/testkube.git",
					Revision: "main",
					AuthType: testsv3.GitAuthTypeHeader,
					TokenFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "git-creds"},
							Key:                  "token",
						},
					},
					UsernameFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.name",
						},
					},
					SshKeyFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{
							ContainerName: "worker",
							Resource:      "limits.cpu",
							Divisor:       resource.MustParse("1m"),
						},
					},
				},
			},
			Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: workflowtriggersv1.FieldOperatorEquals, Value: "3"},
			},
			Wait: &workflowtriggersv1.WorkflowTriggerWait{
				Conditions: &workflowtriggersv1.WorkflowTriggerWaitConditions{
					Items: []workflowtriggersv1.WorkflowTriggerCondition{
						{Type: "Available", Status: &status},
					},
					Timeout: 60,
				},
			},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{Name: "smoke"},
				Delay:    &delay,
			},
		},
	}

	api := MapCRDToAPI(crd)

	assert.Equal(t, "canary", api.Name)
	assert.Equal(t, "prod", api.Namespace)
	assert.True(t, api.Disabled)
	require.NotNil(t, api.Watch)
	assert.Equal(t, "Rollout", api.Watch.Resource.Kind)
	assert.Equal(t, "modified", api.When.Event)
	require.NotNil(t, api.When.Git)
	assert.Equal(t, "https://github.com/kubeshop/testkube.git", api.When.Git.Uri)
	assert.Equal(t, "main", api.When.Git.Revision)
	require.NotNil(t, api.When.Git.AuthType)
	assert.Equal(t, testkube.HEADER_ContentGitAuthType, *api.When.Git.AuthType)
	require.NotNil(t, api.When.Git.TokenFrom)
	require.NotNil(t, api.When.Git.TokenFrom.SecretKeyRef)
	assert.Equal(t, "git-creds", api.When.Git.TokenFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", api.When.Git.TokenFrom.SecretKeyRef.Key)
	require.NotNil(t, api.When.Git.UsernameFrom)
	require.NotNil(t, api.When.Git.UsernameFrom.FieldRef)
	assert.Equal(t, "v1", api.When.Git.UsernameFrom.FieldRef.ApiVersion)
	assert.Equal(t, "metadata.name", api.When.Git.UsernameFrom.FieldRef.FieldPath)
	require.NotNil(t, api.When.Git.SshKeyFrom)
	require.NotNil(t, api.When.Git.SshKeyFrom.ResourceFieldRef)
	assert.Equal(t, "worker", api.When.Git.SshKeyFrom.ResourceFieldRef.ContainerName)
	assert.Equal(t, "limits.cpu", api.When.Git.SshKeyFrom.ResourceFieldRef.Resource)
	assert.Equal(t, "1m", api.When.Git.SshKeyFrom.ResourceFieldRef.Divisor)
	require.Len(t, api.Match, 1)
	assert.Equal(t, "equals", api.Match[0].Operator)
	require.NotNil(t, api.Wait)
	require.NotNil(t, api.Wait.Conditions)
	assert.Equal(t, int32(60), api.Wait.Conditions.Timeout)
	assert.Equal(t, "True", api.Wait.Conditions.Items[0].Status)
	assert.Equal(t, "smoke", api.Run.Workflow.Name)
	assert.Equal(t, "30s", api.Run.Delay)
}

func TestMapAPIToCRD_wrapsSpecAndParsesDelay(t *testing.T) {
	api := testkube.WorkflowTrigger{
		Name:     "canary",
		Disabled: true,
		When: testkube.WorkflowTriggerWhen{
			Event: "created",
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: "main",
				AuthType: ptr(testkube.HEADER_ContentGitAuthType),
				TokenFrom: &testkube.EnvVarSource{
					ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{
						Name:     "git-creds",
						Key:      "token",
						Optional: ptr(true),
					},
				},
				UsernameFrom: &testkube.EnvVarSource{
					FieldRef: &testkube.FieldRef{
						ApiVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
				SshKeyFrom: &testkube.EnvVarSource{
					ResourceFieldRef: &testkube.ResourceFieldRef{
						ContainerName: "runner",
						Resource:      "requests.memory",
						Divisor:       "1Mi",
					},
				},
			},
		},
		Match: []testkube.WorkflowTriggerFieldCondition{{Path: ".spec.image", Operator: "changed"}},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "smoke"},
			Delay:    "10s",
		},
	}

	crd := MapAPIToCRD(api)

	assert.Equal(t, "canary", crd.Name)
	assert.True(t, crd.Spec.Disabled)
	assert.Equal(t, "created", crd.Spec.When.Event)
	require.NotNil(t, crd.Spec.When.Git)
	assert.Equal(t, "https://github.com/kubeshop/testkube.git", crd.Spec.When.Git.Uri)
	assert.Equal(t, "main", crd.Spec.When.Git.Revision)
	assert.Equal(t, testsv3.GitAuthTypeHeader, crd.Spec.When.Git.AuthType)
	require.NotNil(t, crd.Spec.When.Git.TokenFrom)
	require.NotNil(t, crd.Spec.When.Git.TokenFrom.ConfigMapKeyRef)
	assert.NotNil(t, crd.Spec.When.Git.TokenFrom.ConfigMapKeyRef.Optional)
	assert.True(t, *crd.Spec.When.Git.TokenFrom.ConfigMapKeyRef.Optional)
	require.NotNil(t, crd.Spec.When.Git.UsernameFrom)
	require.NotNil(t, crd.Spec.When.Git.UsernameFrom.FieldRef)
	assert.Equal(t, "v1", crd.Spec.When.Git.UsernameFrom.FieldRef.APIVersion)
	assert.Equal(t, "metadata.namespace", crd.Spec.When.Git.UsernameFrom.FieldRef.FieldPath)
	require.NotNil(t, crd.Spec.When.Git.SshKeyFrom)
	require.NotNil(t, crd.Spec.When.Git.SshKeyFrom.ResourceFieldRef)
	assert.Equal(t, "runner", crd.Spec.When.Git.SshKeyFrom.ResourceFieldRef.ContainerName)
	assert.Equal(t, "requests.memory", crd.Spec.When.Git.SshKeyFrom.ResourceFieldRef.Resource)
	assert.Equal(t, "1Mi", crd.Spec.When.Git.SshKeyFrom.ResourceFieldRef.Divisor.String())
	require.Len(t, crd.Spec.Match, 1)
	assert.Equal(t, workflowtriggersv1.FieldOperatorChanged, crd.Spec.Match[0].Operator)
	require.NotNil(t, crd.Spec.Run.Delay)
	assert.Equal(t, 10*time.Second, crd.Spec.Run.Delay.Duration)
}

func TestMapAPIToCRD_invalidDelay_dropsField(t *testing.T) {
	api := testkube.WorkflowTrigger{
		Name: "bad-delay",
		When: testkube.WorkflowTriggerWhen{Event: "created"},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "smoke"},
			Delay:    "not-a-duration",
		},
	}

	crd := MapAPIToCRD(api)

	assert.Nil(t, crd.Spec.Run.Delay, "invalid delay should be dropped rather than panic")
}

func TestMapEnvVarSourceAPIToKube_preservesOptional(t *testing.T) {
	env := &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{
			Name:     "cfg",
			Key:      "token",
			Optional: ptr(true),
		},
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{
			Name:     "sec",
			Key:      "password",
			Optional: ptr(false),
		},
	}

	mapped := mapEnvVarSourceAPIToKube(env)
	require.NotNil(t, mapped)
	require.NotNil(t, mapped.ConfigMapKeyRef)
	require.NotNil(t, mapped.ConfigMapKeyRef.Optional)
	assert.True(t, *mapped.ConfigMapKeyRef.Optional)
	require.NotNil(t, mapped.SecretKeyRef)
	require.NotNil(t, mapped.SecretKeyRef.Optional)
	assert.False(t, *mapped.SecretKeyRef.Optional)
}

func TestMapListCRDToAPI_handlesEmptyAndMulti(t *testing.T) {
	assert.Nil(t, MapListCRDToAPI(nil))

	list := &workflowtriggersv1.WorkflowTriggerList{Items: []workflowtriggersv1.WorkflowTrigger{
		{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
	}}
	out := MapListCRDToAPI(list)
	require.Len(t, out, 2)
	assert.Equal(t, "a", out[0].Name)
	assert.Equal(t, "b", out[1].Name)
}

func ptr[T any](v T) *T {
	return &v
}
