package volume

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openBoth wires the two mounts the way the processor does: the store is the whole
// volume, the inbox is the subPath directory inside it.
func openBoth(t *testing.T) (*Store, *Inbox, string) {
	t.Helper()
	mount := t.TempDir()
	inboxDir := filepath.Join(mount, InboxDir, "exec-1")
	require.NoError(t, os.MkdirAll(inboxDir, 0o777))

	s, reason := OpenStore(mount)
	require.Empty(t, reason)
	require.NotNil(t, s)
	t.Cleanup(func() { s.Close() })

	in, reason := OpenInbox(inboxDir, InboxFor("exec-1"))
	require.Empty(t, reason)
	require.NotNil(t, in)
	t.Cleanup(func() { in.Close() })

	return s, in, mount
}

// Nil plus a reason is the whole fallback mechanism, so every way of not having a
// volume has to produce it rather than an error the caller would have to classify.
func TestOpenReportsWhyThereIsNoVolume(t *testing.T) {
	t.Run("store not configured", func(t *testing.T) {
		s, reason := OpenStore("")
		assert.Nil(t, s)
		assert.NotEmpty(t, reason)
	})

	t.Run("store mount is missing", func(t *testing.T) {
		s, reason := OpenStore(filepath.Join(t.TempDir(), "absent"))
		assert.Nil(t, s)
		assert.Contains(t, reason, "cannot read")
	})

	t.Run("inbox not configured", func(t *testing.T) {
		in, reason := OpenInbox("", InboxFor("exec-1"))
		assert.Nil(t, in)
		assert.NotEmpty(t, reason)
	})

	t.Run("inbox has no name", func(t *testing.T) {
		in, reason := OpenInbox(t.TempDir(), "")
		assert.Nil(t, in)
		assert.NotEmpty(t, reason)
	})
}

// A read-only mount is the case the probe exists for: without it the first failure
// would come after copying a tree that can run to gigabytes.
func TestOpenInboxProbesInsteadOfTrustingTheMount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not deny writes the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores the permission bits this relies on")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	in, reason := OpenInbox(dir, InboxFor("exec-1"))

	assert.Nil(t, in)
	assert.Contains(t, reason, "cannot write")
}

// The probe must not survive itself: anything left in the inbox would be copied into a
// later entry, or counted by the sweep as work in progress.
func TestOpenInboxLeavesNothingBehind(t *testing.T) {
	_, _, mount := openBoth(t)

	entries, err := os.ReadDir(filepath.Join(mount, InboxDir, "exec-1"))

	require.NoError(t, err)
	assert.Empty(t, entries)
}

// The pointer is composed from the volume root even though the inbox writes into a
// flat subPath mount - a reader mounts the whole volume, so that is the name it needs.
func TestCommitNamesTheEntryFromTheVolumeRoot(t *testing.T) {
	s, in, mount := openBoth(t)
	dir, staged, err := in.Stage()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data", "dep"), []byte("bytes"), 0o666))

	p, err := in.Commit(staged, 5)

	require.NoError(t, err)
	require.NoError(t, ValidatePath(p.Path))
	assert.True(t, strings.HasPrefix(p.Path, InboxDir+"/exec-1/"), "got %q", p.Path)
	assert.EqualValues(t, 5, p.Size)

	body, err := os.ReadFile(filepath.Join(mount, filepath.FromSlash(p.Path), EntryRoot, "data", "dep"))
	require.NoError(t, err)
	assert.Equal(t, "bytes", string(body))

	// And the reader reaches the entry by that name, with the mirrored tree inside it.
	root, err := s.OpenEntry(p)
	require.NoError(t, err)
	defer root.Close()
	f, err := root.Open(EntryRoot + "/data/dep")
	require.NoError(t, err)
	f.Close()
}

// A .tmp left beside the entry is a copy that looks like it is still running, which is
// exactly what the sweep is told to leave alone.
func TestCommitLeavesNoStagedDirectory(t *testing.T) {
	_, in, mount := openBoth(t)
	_, staged, err := in.Stage()
	require.NoError(t, err)

	_, err = in.Commit(staged, 0)
	require.NoError(t, err)

	entries, err := os.ReadDir(filepath.Join(mount, InboxDir, "exec-1"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.False(t, strings.HasSuffix(entries[0].Name(), ".tmp"))
}

// The pointer came out of an object a pod wrote, so OpenEntry is the last place that
// can refuse it - and the only one that turns it into a directory.
func TestOpenEntryValidatesThePointer(t *testing.T) {
	s, _, mount := openBoth(t)
	require.NoError(t, os.MkdirAll(filepath.Join(mount, "secret"), 0o777))

	_, err := s.OpenEntry(Pointer{Path: "../secret"})

	assert.Error(t, err)
}

func TestOpenEntryReportsAMissingEntry(t *testing.T) {
	s, _, _ := openBoth(t)

	_, err := s.OpenEntry(Pointer{Path: "inbox/exec-1/absent"})

	assert.ErrorIs(t, err, os.ErrNotExist)
}

// Every path that does not publish a pointer has to discard, or the entry is
// unreachable and invisible until the volume fills.
func TestDiscardRemovesAnUnpublishedEntry(t *testing.T) {
	_, in, mount := openBoth(t)
	dir, staged, err := in.Stage()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "deep", "nested"), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deep", "nested", "f"), []byte("x"), 0o666))
	p, err := in.Commit(staged, 1)
	require.NoError(t, err)

	in.Discard(p)

	_, err = os.Stat(filepath.Join(mount, filepath.FromSlash(p.Path)))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestDiscardStagedRemovesAnAbandonedCopy(t *testing.T) {
	_, in, mount := openBoth(t)
	dir, staged, err := in.Stage()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "half"), []byte("x"), 0o666))

	in.DiscardStaged(staged)

	entries, err := os.ReadDir(filepath.Join(mount, InboxDir, "exec-1"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDiscardRefusesAPointerThisInboxDidNotWrite(t *testing.T) {
	_, in, mount := openBoth(t)
	other := filepath.Join(mount, InboxDir, "exec-2")
	require.NoError(t, os.MkdirAll(other, 0o777))
	victim := filepath.Join(other, "theirs")
	require.NoError(t, os.MkdirAll(victim, 0o777))

	in.Discard(Pointer{Path: "inbox/exec-2/theirs"})

	assert.DirExists(t, victim)
}

func TestDiscardIsSafeToRepeatAndToMisuse(t *testing.T) {
	_, in, _ := openBoth(t)

	assert.NotPanics(t, func() {
		in.Discard(Pointer{})
		in.Discard(Pointer{Path: "inbox/exec-1/absent"})
		in.Discard(Pointer{Path: "../escape"})
		in.DiscardStaged("")
		var nilInbox *Inbox
		nilInbox.Discard(Pointer{Path: "inbox/exec-1/x"})
		nilInbox.DiscardStaged("x")
	})
}

// Two cached steps in one execution share an inbox, so a predictable name would have
// one overwrite the other's entry.
func TestStagedNamesDoNotCollide(t *testing.T) {
	_, in, _ := openBoth(t)

	_, first, err := in.Stage()
	require.NoError(t, err)
	_, second, err := in.Stage()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}
