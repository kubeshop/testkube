package localrunner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestRewriteWorkflowWithSourcePreservesInputAndGitMount(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{}
	workflow.Spec.Content = &testworkflowsv1.Content{Git: &testworkflowsv1.ContentGit{Uri: "https://example.invalid/repo.git", MountPath: "/workspace"}}

	mount, err := ResolveSourceMount(workflow, "")
	require.NoError(t, err)
	assert.Equal(t, "/workspace", mount)
	rewritten, err := RewriteWorkflowWithSource(workflow, "http://source.example/token.tar.gz", mount)
	require.NoError(t, err)
	assert.NotSame(t, workflow, rewritten)
	assert.NotNil(t, workflow.Spec.Content.Git, "input workflow must stay untouched")
	assert.Nil(t, rewritten.Spec.Content.Git)
	require.Len(t, rewritten.Spec.Content.Tarball, 1)
	assert.Equal(t, "/workspace", rewritten.Spec.Content.Tarball[0].Path)
	assert.Equal(t, "http://source.example/token.tar.gz", rewritten.Spec.Content.Tarball[0].Url)
	require.NotNil(t, rewritten.Spec.Content.Tarball[0].Mount)
	assert.True(t, *rewritten.Spec.Content.Tarball[0].Mount)
}

func TestRewriteWorkflowWithSourceRejectsMountCollision(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{}
	workflow.Spec.Content = &testworkflowsv1.Content{Tarball: []testworkflowsv1.ContentTarball{{Url: "https://example.invalid/existing.tgz", Path: "/data/repo"}}}
	_, err := RewriteWorkflowWithSource(workflow, "http://source.example/token.tar.gz", "/data/repo")
	require.Error(t, err)
	assert.True(t, IsUsageError(err))
	assert.ErrorContains(t, err, "spec.content.tarball path")
}

func TestResolveSourceMountRejectsUnsafePath(t *testing.T) {
	_, err := ResolveSourceMount(&testworkflowsv1.TestWorkflow{}, "relative/path")
	require.Error(t, err)
	assert.True(t, IsUsageError(err))
	_, err = ResolveSourceMount(&testworkflowsv1.TestWorkflow{}, "/data/../repo")
	require.Error(t, err)
}

func TestRelayResourcesUseExactLabels(t *testing.T) {
	labels, err := Labels("local-demo", "source-relay")
	require.NoError(t, err)
	pod := relayPod("relay", DefaultNamespace, labels)
	service := relayService("relay", DefaultNamespace, labels)
	assert.Equal(t, labels, pod.Labels)
	assert.Equal(t, labels, service.Labels)
	assert.Equal(t, labels, service.Spec.Selector)
	assert.Equal(t, int64(3600), *pod.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, sourceRelayImage, pod.Spec.Containers[0].Image)
}

func TestLocalObjectNameRetainsRunSuffixForLongWorkflowNames(t *testing.T) {
	firstRunID := "local-" + strings.Repeat("same-workflow-prefix-", 3) + "aaaabbbbcccc"
	secondRunID := "local-" + strings.Repeat("same-workflow-prefix-", 3) + "ddddeeeeffff"
	first := localObjectName("local-source", firstRunID)
	second := localObjectName("local-source", secondRunID)
	assert.LessOrEqual(t, len(first), 63)
	assert.LessOrEqual(t, len(second), 63)
	assert.NotEqual(t, first, second)
	assert.True(t, strings.HasSuffix(first, "aaaabbbbcccc"))
	assert.True(t, strings.HasSuffix(second, "ddddeeeeffff"))
}
