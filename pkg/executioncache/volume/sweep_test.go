package volume

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inboxAged creates an inbox holding one entry, last written the given time ago.
func inboxAged(t *testing.T, root, id string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, InboxDir, id)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "entry", EntryRoot), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entry", EntryRoot, "f"), []byte("x"), 0o666))

	when := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(dir, when, when))
	return dir
}

func TestSweepRemovesInboxesPastRetention(t *testing.T) {
	root := t.TempDir()
	old := inboxAged(t, root, "exec-old", 48*time.Hour)
	fresh := inboxAged(t, root, "exec-fresh", time.Minute)

	s := &Sweeper{Root: root, Retention: 24 * time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	assert.NoDirExists(t, old)
	assert.DirExists(t, fresh, "an inbox inside the window must survive")
}

// A volume nothing has cached to yet is the normal first state, the same way a cold
// bucket is - not something to report as a fault every interval.
func TestSweepTreatsAColdVolumeAsNormal(t *testing.T) {
	s := &Sweeper{Root: t.TempDir(), Retention: time.Hour}

	assert.NoError(t, s.Sweep(context.Background()))
}

func TestSweepDoesNothingWithoutRetention(t *testing.T) {
	root := t.TempDir()
	dir := inboxAged(t, root, "exec-old", 1000*time.Hour)

	s := &Sweeper{Root: root}
	require.NoError(t, s.Sweep(context.Background()))

	assert.DirExists(t, dir, "retention of zero must not be read as 'expire everything'")
}

// The leader coordinator waits for the task to return before handing leadership over,
// so a sweep that ignored cancellation would stall the handover for a whole walk.
func TestSweepStopsWhenCancelled(t *testing.T) {
	root := t.TempDir()
	dir := inboxAged(t, root, "exec-old", 48*time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &Sweeper{Root: root, Retention: time.Hour}
	require.NoError(t, s.Sweep(ctx))

	assert.DirExists(t, dir, "a cancelled sweep should stop rather than finish the walk")
}

// An entry is reachable only through its pointer, so the whole inbox goes - including
// an abandoned staging directory a save never committed.
func TestSweepRemovesAbandonedStagingDirectories(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-dead")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "halfway.tmp", EntryRoot), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "halfway.tmp", EntryRoot, "partial"), []byte("x"), 0o666))
	when := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(dir, when, when))

	s := &Sweeper{Root: root, Retention: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	assert.NoDirExists(t, dir)
}

func TestSweepIgnoresStrayFilesBesideTheInboxes(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, InboxDir), 0o777))
	stray := filepath.Join(root, InboxDir, "not-a-directory")
	require.NoError(t, os.WriteFile(stray, []byte("x"), 0o666))

	s := &Sweeper{Root: root, Retention: time.Nanosecond}
	require.NoError(t, s.Sweep(context.Background()))

	assert.FileExists(t, stray)
}

func TestRunSweepsImmediatelyAndStops(t *testing.T) {
	root := t.TempDir()
	old := inboxAged(t, root, "exec-old", 48*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	s := &Sweeper{Root: root, Retention: time.Hour, Interval: time.Hour}

	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	assert.Eventually(t, func() bool {
		_, err := os.Stat(old)
		return os.IsNotExist(err)
	}, 2*time.Second, 10*time.Millisecond, "the first sweep must not wait out an interval")

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err, "losing leadership is a handover, not a failure")
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}

// An inbox is made before its pod starts and its mtime only moves when something is
// staged inside it, so an execution that has cached nothing yet - or runs one long step
// - looks expired however alive it is. Sweeping it unlinks a directory the running pod
// still holds through its subPath: the writes land on an unreachable inode, and the
// pointer that execution publishes names a path that is gone.
func TestSweepKeepsAnInboxWithAFreshLease(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-live")
	require.NoError(t, os.MkdirAll(dir, 0o777))
	age(t, dir)
	require.NoError(t, TouchLease(root, InboxDir+"/exec-live"))

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	_, err := os.Stat(dir)
	assert.NoError(t, err, "a running execution still writes here")
}

// A lease that stops being refreshed has to stop protecting, or an agent that died
// mid-execution would pin that inbox for good and fill the volume.
func TestSweepRemovesAnInboxWhoseLeaseWentStale(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-gone")
	require.NoError(t, os.MkdirAll(dir, 0o777))
	require.NoError(t, TouchLease(root, InboxDir+"/exec-gone"))
	stale := filepath.Join(dir, LeaseName)
	age(t, stale)
	age(t, dir)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "a lease nobody refreshes must not pin the volume")
}

// A refresh arriving after a sweep must not rebuild an inbox nothing is mounted at.
func TestTouchLeaseDoesNotCreateTheInbox(t *testing.T) {
	root := t.TempDir()

	err := TouchLease(root, InboxDir+"/never-existed")

	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(root, InboxDir, "never-existed"))
	assert.True(t, os.IsNotExist(statErr))
}

// The name decides where a file is written, so it is checked rather than trusted.
func TestTouchLeaseRefusesAnythingThatIsNotAnInboxName(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", "exec-1", "inbox", "inbox/../../etc", "inbox/a/b", "/etc"} {
		assert.Error(t, TouchLease(root, name), "name %q", name)
	}
}

// age backdates a path well past any retention a test sets.
func age(t *testing.T, name string) {
	t.Helper()
	when := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(name, when, when))
}

// The step's own command holds the inbox mount - the save stage is pure, so it merges
// into that container - which means a workflow can write .lease itself and date it a
// century ahead. Treating a negative age as fresh would keep that inbox for good, and a
// few of them would fill a volume every execution in the cluster shares.
func TestSweepRemovesAnInboxWhoseLeaseIsDatedInTheFuture(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-squatter")
	require.NoError(t, os.MkdirAll(dir, 0o777))
	require.NoError(t, TouchLease(root, InboxDir+"/exec-squatter"))
	ahead := time.Now().Add(100 * 365 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, LeaseName), ahead, ahead))
	age(t, dir)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "a lease nobody can outlive must not pin the volume")
}

// The inbox's own mtime is reachable the same way, so it is bounded the same way.
func TestSweepRemovesAnInboxDatedInTheFuture(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-squatter")
	require.NoError(t, os.MkdirAll(dir, 0o777))
	ahead := time.Now().Add(100 * 365 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(dir, ahead, ahead))

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

// A lease written moments ago may read as marginally ahead, because the agent and
// whatever serves the volume keep their own clocks. That must not sweep a live inbox.
func TestSweepKeepsAnInboxWhoseLeaseIsBarelyAhead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, InboxDir, "exec-live")
	require.NoError(t, os.MkdirAll(dir, 0o777))
	require.NoError(t, TouchLease(root, InboxDir+"/exec-live"))
	ahead := time.Now().Add(maxClockSkew / 2)
	require.NoError(t, os.Chtimes(filepath.Join(dir, LeaseName), ahead, ahead))
	age(t, dir)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	require.NoError(t, s.Sweep(context.Background()))

	_, err := os.Stat(dir)
	assert.NoError(t, err, "a small clock difference is not a squatter")
}

// Continuing past an inbox it cannot remove is what keeps one stuck entry from
// stranding every inbox after it. Reporting what it could not take is what keeps that
// from happening in silence, with the volume filling and nothing said - the caller logs
// whatever Sweep returns.
func TestSweepReportsWhatItCouldNotRemoveAndKeepsGoing(t *testing.T) {
	root := t.TempDir()
	stuck := inboxAged(t, root, "exec-stuck", 48*time.Hour)
	other := inboxAged(t, root, "exec-other", 48*time.Hour)

	// Stands in for a directory the agent cannot unlink through: an open handle on
	// Windows, a mode the sweeping user does not hold on Linux.
	blocker, err := os.Open(filepath.Join(stuck, "entry", EntryRoot, "f"))
	require.NoError(t, err)
	require.NoError(t, os.Chmod(filepath.Join(stuck, "entry", EntryRoot), 0o500))
	t.Cleanup(func() {
		blocker.Close()
		_ = os.Chmod(filepath.Join(stuck, "entry", EntryRoot), 0o777)
	})

	s := &Sweeper{Root: root, Retention: time.Hour}
	err = s.Sweep(context.Background())

	// The one it could take is gone whatever happened to the other.
	_, otherErr := os.Stat(other)
	assert.True(t, os.IsNotExist(otherErr), "a stuck inbox must not strand the ones after it")

	if _, stillThere := os.Stat(stuck); stillThere == nil {
		require.Error(t, err, "an inbox it could not remove has to be reported")
		assert.Contains(t, err.Error(), "exec-stuck")
	}
}
