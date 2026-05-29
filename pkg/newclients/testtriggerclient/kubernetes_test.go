package testtriggerclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testtriggersclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testtriggers/v1"
)

func TestKubernetesTestTriggerClient_CreateMapsExtendedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := testtriggersclientv1.NewMockInterface(ctrl)
	client := NewKubernetesTestTriggerClient(mockClient)

	trigger := buildAPITrigger()

	mockClient.EXPECT().Create(gomock.Any()).DoAndReturn(func(crd *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error) {
		require.Len(t, crd.Spec.Match, 1)
		assert.Equal(t, string(trigger.Match[0].Operator), string(crd.Spec.Match[0].Operator))
		assert.Equal(t, trigger.Match[0].Path, crd.Spec.Match[0].Path)
		assert.Equal(t, trigger.Match[0].Value, crd.Spec.Match[0].Value)
		require.NotNil(t, crd.Spec.ContentSelector)
		require.NotNil(t, crd.Spec.ContentSelector.Git)
		assert.Equal(t, trigger.ContentSelector.Git.Uri, crd.Spec.ContentSelector.Git.Uri)
		assert.Equal(t, trigger.ContentSelector.Git.Branches, crd.Spec.ContentSelector.Git.Branches)
		assert.Equal(t, trigger.ContentSelector.Git.Username, crd.Spec.ContentSelector.Git.Username)
		assert.Equal(t, trigger.ContentSelector.Git.Token, crd.Spec.ContentSelector.Git.Token)
		assert.Equal(t, trigger.ContentSelector.Git.SshKey, crd.Spec.ContentSelector.Git.SshKey)
		require.NotNil(t, trigger.ContentSelector.Git.AuthType)
		assert.Equal(t, string(*trigger.ContentSelector.Git.AuthType), string(crd.Spec.ContentSelector.Git.AuthType))
		require.NotNil(t, crd.Spec.ContentSelector.Git.UsernameFrom)
		require.NotNil(t, crd.Spec.ContentSelector.Git.TokenFrom)
		require.NotNil(t, crd.Spec.ContentSelector.Git.SshKeyFrom)
		assert.Equal(t, trigger.ContentSelector.Git.UsernameFrom.SecretKeyRef.Key, crd.Spec.ContentSelector.Git.UsernameFrom.SecretKeyRef.Key)
		assert.Equal(t, trigger.ContentSelector.Git.TokenFrom.SecretKeyRef.Key, crd.Spec.ContentSelector.Git.TokenFrom.SecretKeyRef.Key)
		assert.Equal(t, trigger.ContentSelector.Git.SshKeyFrom.SecretKeyRef.Key, crd.Spec.ContentSelector.Git.SshKeyFrom.SecretKeyRef.Key)
		assert.Equal(t, trigger.ContentSelector.Git.Paths, crd.Spec.ContentSelector.Git.Paths)
		return crd, nil
	})

	err := client.Create(context.Background(), "", trigger)
	require.NoError(t, err)
}

func TestKubernetesTestTriggerClient_UpdateMapsExtendedFieldsAndPreservesMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := testtriggersclientv1.NewMockInterface(ctrl)
	client := NewKubernetesTestTriggerClient(mockClient)

	trigger := buildAPITrigger()
	existing := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:            trigger.Name,
			Namespace:       trigger.Namespace,
			ResourceVersion: "12345",
		},
	}

	mockClient.EXPECT().Get(trigger.Name, trigger.Namespace).Return(existing, nil)
	mockClient.EXPECT().Update(gomock.Any()).DoAndReturn(func(crd *testtriggersv1.TestTrigger) (*testtriggersv1.TestTrigger, error) {
		assert.Equal(t, existing.ResourceVersion, crd.ResourceVersion)
		require.Len(t, crd.Spec.Match, 1)
		assert.Equal(t, trigger.Match[0].Path, crd.Spec.Match[0].Path)
		require.NotNil(t, crd.Spec.ContentSelector)
		require.NotNil(t, crd.Spec.ContentSelector.Git)
		assert.Equal(t, trigger.ContentSelector.Git.Uri, crd.Spec.ContentSelector.Git.Uri)
		assert.Equal(t, trigger.ContentSelector.Git.Token, crd.Spec.ContentSelector.Git.Token)
		require.NotNil(t, trigger.ContentSelector.Git.AuthType)
		assert.Equal(t, string(*trigger.ContentSelector.Git.AuthType), string(crd.Spec.ContentSelector.Git.AuthType))
		return crd, nil
	})

	err := client.Update(context.Background(), "", trigger)
	require.NoError(t, err)
}

func buildAPITrigger() testkube.TestTrigger {
	resource := testkube.CONTENT_TestTriggerResources
	action := testkube.RUN_TestTriggerActions
	execution := testkube.TESTWORKFLOW_TestTriggerExecutions
	concurrency := testkube.ALLOW_TestTriggerConcurrencyPolicies
	authType := testkube.BASIC_ContentGitAuthType

	return testkube.TestTrigger{
		Name:             "git-trigger",
		Namespace:        "testkube",
		Resource:         &resource,
		Event:            string(testtriggersv1.TestTriggerEventModified),
		ResourceSelector: &testkube.TestTriggerSelector{},
		Match: []testkube.TestTriggerFieldCondition{
			{
				Path:     ".spec.replicas",
				Operator: testkube.TestTriggerFieldOperatorChangedTo,
				Value:    "2",
			},
		},
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Branches: []string{"main"},
				Username: "git-user",
				Token:    "token-value",
				SshKey:   "ssh-private-key",
				AuthType: &authType,
				Paths:    []string{"pkg/triggers"},
				UsernameFrom: &testkube.EnvVarSource{
					SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{
						Name: "username",
						Key:  "username",
					},
				},
				TokenFrom: &testkube.EnvVarSource{
					SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{
						Name: "token",
						Key:  "token",
					},
				},
				SshKeyFrom: &testkube.EnvVarSource{
					SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{
						Name: "sshKey",
						Key:  "sshKey",
					},
				},
			},
		},
		Action:            &action,
		Execution:         &execution,
		TestSelector:      &testkube.TestTriggerSelector{},
		ConcurrencyPolicy: &concurrency,
	}
}
