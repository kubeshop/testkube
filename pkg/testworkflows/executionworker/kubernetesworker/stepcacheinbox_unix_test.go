//go:build unix

package kubernetesworker

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// The shared parent is made once, by whichever agent reaches the volume first, and
// MkdirAll's mode is filtered by that agent's umask. Left at 0755 it belongs to that
// agent's user alone, and every other agent on the same backing volume - a runner
// beside an api deployment, or one running as a different UID - can then neither add an
// inbox nor remove an expired one. Saves fall back to the object store and nothing is
// ever reclaimed, on a volume the whole cluster shares.
func TestPrepareStepCacheInboxOpensTheSharedParentToo(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	root := t.TempDir()
	w := &worker{config: Config{
		StepCacheVolume:          &testworkflowconfig.StepCacheVolumeConfig{ClaimName: "step-cache", ID: "vol1"},
		StepCacheVolumeLocalPath: root,
	}}

	w.prepareStepCacheInbox(cacheClaimBundle("step-cache"), "exec-1")

	for _, name := range []string{
		filepath.Join(root, volume.InboxDir),
		filepath.Join(root, volume.InboxDir, "exec-1"),
	} {
		info, err := os.Stat(name)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(cacheInboxMode), info.Mode().Perm(),
			"%s is a directory another agent could not write", name)
	}
}
