//go:build unix

package volume

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every directory written onto the shared volume has to stay writable by whoever sweeps
// it. The step container creates these under its own umask and as whatever user the
// workflow chose; the agent removes them as another. A directory's write bit is what
// permits unlinking what is inside it, so one directory left at 0755 is a subtree the
// agent can never reclaim - and because the sweep removes whole inboxes, it strands
// everything beside it too.
//
// Unix-only because the umask is the whole point: this has to fail when the mode passed
// to Mkdir is all that is relied on.
func TestSaveLeavesEveryDirectoryWritable(t *testing.T) {
	// The usual setting in a container, which strips the group and other write bits
	// from whatever mode Mkdir is given.
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	src := posixDir(t)
	write(t, filepath.FromSlash(src+"/nested/deep/dep"), "x")

	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))
	require.NoError(t, os.Chmod(staging, 0o777))

	_, _, err := SaveTree(staging, []string{src}, CopyLimits{})
	require.NoError(t, err)

	require.NoError(t, filepath.Walk(staging, func(name string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		assert.Equal(t, os.FileMode(SharedDirMode), info.Mode().Perm(),
			"%s is a directory the sweep could not unlink through", name)
		return nil
	}))
}
