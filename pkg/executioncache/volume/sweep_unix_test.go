//go:build unix

package volume

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lease sits inside a directory the step's own command can write, so a workflow can
// replace it with a symlink pointing at itself: every Stat of it then answers ELOOP.
//
// Returning that error stopped the sweep at this inbox on every pass, so nothing after
// it was ever reclaimed and the shared volume filled - one workflow able to wedge the
// cache for every execution in the cluster. A lease that cannot be read is simply not
// believed, so the inbox expires by mtime the ordinary way and the sweep carries on.
//
// Unix-only because creating the self-referential link is the whole setup.
func TestSweepContinuesWhenALeaseCannotBeRead(t *testing.T) {
	root := t.TempDir()
	wedged := inboxAged(t, root, "exec-wedged", 48*time.Hour)
	other := inboxAged(t, root, "exec-other", 48*time.Hour)
	require.NoError(t, os.Symlink(LeaseName, filepath.Join(wedged, LeaseName)))

	// The symlink is the last thing written, so put the directory back out of the
	// retention window.
	age(t, wedged)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	err := s.Sweep(context.Background())

	_, otherErr := os.Stat(other)
	assert.True(t, os.IsNotExist(otherErr), "an unreadable lease must not strand the inboxes after it")
	_, wedgedErr := os.Stat(wedged)
	assert.True(t, os.IsNotExist(wedgedErr), "and the inbox it could not vouch for expires the ordinary way")

	require.Error(t, err, "the operator still has to hear that a lease could not be read")
	assert.Contains(t, err.Error(), "exec-wedged")
}

// The inbox is writable by the step, so the step owns what .lease is. os.Chtimes and
// os.OpenFile both follow the final symlink, so a workflow pointing .lease at an
// absolute path would have the agent stamp - or create - a file inside its own
// container: a process the step cannot reach and that holds far more privilege.
func TestTouchLeaseDoesNotFollowALinkOutOfTheInbox(t *testing.T) {
	root := t.TempDir()
	inbox := filepath.Join(root, InboxDir, "exec-1")
	require.NoError(t, os.MkdirAll(inbox, 0o777))

	// Somewhere the agent can reach and the step cannot, standing in for anything in
	// the agent's own filesystem.
	outside := filepath.Join(t.TempDir(), "agent-file")
	require.NoError(t, os.WriteFile(outside, []byte("untouched"), 0o666))
	before, err := os.Stat(outside)
	require.NoError(t, err)

	require.NoError(t, os.Symlink(outside, filepath.Join(inbox, LeaseName)))

	require.NoError(t, TouchLease(root, InboxDir+"/exec-1"))

	after, err := os.Stat(outside)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "the agent must not stamp a file the step named")

	// The link is replaced by a lease of this process's own, so the inbox is not left
	// pinned by a timestamp the step controls.
	lease, err := os.Lstat(filepath.Join(inbox, LeaseName))
	require.NoError(t, err)
	assert.Zero(t, lease.Mode()&os.ModeSymlink, "a lease is a file this process writes")
}

// And a lease that is a link is not believed by the sweep either, so it cannot pin an
// inbox by naming something with a convenient mtime.
func TestSweepDoesNotBelieveASymlinkedLease(t *testing.T) {
	root := t.TempDir()
	dir := inboxAged(t, root, "exec-1", 48*time.Hour)

	fresh := filepath.Join(t.TempDir(), "fresh")
	require.NoError(t, os.WriteFile(fresh, []byte("now"), 0o666))
	require.NoError(t, os.Symlink(fresh, filepath.Join(dir, LeaseName)))
	age(t, dir)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	err := s.Sweep(context.Background())

	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "a symlinked lease must not keep the inbox")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

// Both of these are read or written by agents that did not create them: an api and a
// runner sharing a volume hold separate leader elections and run as whatever user each
// was given. A mode left to the umask makes them private to whoever got there first.
//
// The identity is the worse of the two - an agent that cannot read it gets none, and
// without one the volume is turned off for it entirely, so a single strict umask would
// take the feature away from every other agent on the volume.
func TestAgentWrittenFilesStayReadableUnderAStrictUmask(t *testing.T) {
	previous := syscall.Umask(0o077)
	defer syscall.Umask(previous)

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, InboxDir, "exec-1"), 0o777))

	_, err := EnsureID(root)
	require.NoError(t, err)
	require.NoError(t, TouchLease(root, InboxDir+"/exec-1"))

	for _, name := range []string{
		filepath.Join(root, IDName),
		filepath.Join(root, InboxDir, "exec-1", LeaseName),
	} {
		info, statErr := os.Stat(name)
		require.NoError(t, statErr, name)
		assert.Equal(t, fs.FileMode(SharedFileMode), info.Mode().Perm(),
			"%s is one another agent could not use", name)
	}
}

// The step holds its own inbox's mount, so it can create .lease itself - as whatever
// user the workflow runs, under its own umask. A lease left unwritable is one the agent
// can never renew, so it goes stale while the execution is still running and the sweep
// takes the inbox from under its pod. Unlinking it needs the write bit on the inbox
// rather than ownership of the file, so the renewal replaces it.
//
// Unix-only because the mode is the whole point.
func TestTouchLeaseReplacesALeaseItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write any mode, so there is nothing to replace")
	}

	root := t.TempDir()
	inbox := filepath.Join(root, InboxDir, "exec")
	require.NoError(t, os.MkdirAll(inbox, SharedDirMode))
	require.NoError(t, os.Chmod(inbox, SharedDirMode))

	// What the step left behind: a lease nothing else may write.
	lease := filepath.Join(inbox, LeaseName)
	require.NoError(t, os.WriteFile(lease, []byte("theirs\n"), 0o400))
	require.NoError(t, os.Chmod(lease, 0o000))

	require.NoError(t, TouchLease(root, InboxFor("exec")),
		"a lease the agent cannot write must be replaced, not given up on")

	info, err := os.Stat(lease)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), info.ModTime(), time.Minute,
		"the replacement has to carry a fresh timestamp, which is the whole renewal")
}
