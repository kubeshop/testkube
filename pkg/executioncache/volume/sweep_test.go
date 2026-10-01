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
