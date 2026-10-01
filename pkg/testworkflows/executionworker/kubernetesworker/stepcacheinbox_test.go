package kubernetesworker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// kubelet creates a missing subPath directory itself, but owned by root and with the
// volume root's mode, and fsGroup is not applied to a multi-writer volume - so a step
// running as another user could not write to its own inbox, and would fall back to the
// object store on every save with nothing to say why. An existing directory is taken as
// it is, so the agent makes it first.
func TestPrepareStepCacheInboxMakesThisExecutionsInbox(t *testing.T) {
	root := t.TempDir()
	w := &worker{config: Config{
		StepCacheVolume:          &testworkflowconfig.StepCacheVolumeConfig{ClaimName: "step-cache"},
		StepCacheVolumeLocalPath: root,
	}}

	w.prepareStepCacheInbox("exec-1")

	info, err := os.Stat(filepath.Join(root, "inbox", "exec-1"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o777), info.Mode().Perm(),
			"the agent cannot know which user the step runs as")
	}
}

// Making the shared parent here is what keeps it away from kubelet, which before
// Kubernetes v1.37 fails a container that loses a race to create one.
func TestPrepareStepCacheInboxMakesTheSharedParent(t *testing.T) {
	root := t.TempDir()
	w := &worker{config: Config{
		StepCacheVolume:          &testworkflowconfig.StepCacheVolumeConfig{ClaimName: "step-cache"},
		StepCacheVolumeLocalPath: root,
	}}

	w.prepareStepCacheInbox("exec-1")
	w.prepareStepCacheInbox("exec-2")

	for _, id := range []string{"exec-1", "exec-2"} {
		_, err := os.Stat(filepath.Join(root, "inbox", id))
		require.NoError(t, err, "a second execution must not be refused the parent the first made")
	}
}

// Without a volume there is nothing to prepare, and the local path alone is not one:
// it keeps its default whether or not a claim was configured.
func TestPrepareStepCacheInboxDoesNothingWithoutAVolume(t *testing.T) {
	root := t.TempDir()
	w := &worker{config: Config{StepCacheVolumeLocalPath: root}}

	w.prepareStepCacheInbox("exec-1")

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// A volume the agent cannot reach is not worth failing an execution over: kubelet still
// makes the directory, and the pod-side probe still falls back to the object store.
func TestPrepareStepCacheInboxSurvivesAnUnusableVolume(t *testing.T) {
	w := &worker{config: Config{
		StepCacheVolume:          &testworkflowconfig.StepCacheVolumeConfig{ClaimName: "step-cache"},
		StepCacheVolumeLocalPath: filepath.Join(t.TempDir(), "not-mounted", "x\x00y"),
	}}

	assert.NotPanics(t, func() { w.prepareStepCacheInbox("exec-1") })
}
