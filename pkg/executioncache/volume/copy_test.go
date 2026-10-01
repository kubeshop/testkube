package volume

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, name, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o777))
	require.NoError(t, os.WriteFile(name, []byte(body), 0o666))
}

// entryFrom builds a staged entry holding the given declared paths, the way a save
// does, and returns a reader over its mirrored tree.
func entryFrom(t *testing.T, paths []string) *os.Root {
	t.Helper()
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))
	_, err := SaveTree(staging, paths, CopyLimits{})
	require.NoError(t, err)
	root, err := os.OpenRoot(staging)
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })
	return root
}

func TestSaveAndRestoreRoundTripATree(t *testing.T) {
	skipUnlessPosix(t)
	src := t.TempDir()
	write(t, filepath.Join(src, "pkg", "a.js"), "alpha")
	write(t, filepath.Join(src, "pkg", "nested", "b.js"), "beta")

	entry := entryFrom(t, []string{src})
	dest := t.TempDir()
	require.NoError(t, os.RemoveAll(dest))

	// Restoring writes to the absolute path the entry mirrors, so the destination is
	// the source path itself - which is what a restore into a fresh pod does.
	require.NoError(t, os.RemoveAll(src))
	wrote, err := RestoreTree(entry, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.True(t, wrote)
	body, err := os.ReadFile(filepath.Join(src, "pkg", "nested", "b.js"))
	require.NoError(t, err)
	assert.Equal(t, "beta", string(body))
}

// An entry may have been written by another workflow under an environment-scoped
// cache, so anything outside the declared paths is skipped rather than written
// somewhere the step never asked for and the cleanup would not reach.
func TestRestoreSkipsWhatTheStepDidNotDeclare(t *testing.T) {
	skipUnlessPosix(t)
	declared := t.TempDir()
	smuggled := t.TempDir()
	write(t, filepath.Join(declared, "wanted"), "yes")
	write(t, filepath.Join(smuggled, "unwanted"), "no")

	// The entry carries both, but the restore only declares one.
	entry := entryFrom(t, []string{declared, smuggled})
	require.NoError(t, os.RemoveAll(declared))
	require.NoError(t, os.RemoveAll(smuggled))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(declared, "wanted"))
	assert.NoFileExists(t, filepath.Join(smuggled, "unwanted"), "an undeclared path must not be restored")
}

func TestRestoreStopsAtTheSizeLimit(t *testing.T) {
	skipUnlessPosix(t)
	src := t.TempDir()
	write(t, filepath.Join(src, "big"), "0123456789")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxTotalBytes: 4})

	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestRestoreStopsAtTheEntryLimit(t *testing.T) {
	skipUnlessPosix(t)
	src := t.TempDir()
	write(t, filepath.Join(src, "one"), "a")
	write(t, filepath.Join(src, "two"), "b")
	write(t, filepath.Join(src, "three"), "c")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxEntries: 2})

	assert.ErrorIs(t, err, ErrTooManyEntries)
}

// A step may declare a cache path it never creates. That is a smaller entry, not a
// failure - the same way an unmatched hash_files glob is a miss rather than an error.
func TestSaveSkipsAPathThatWasNeverCreated(t *testing.T) {
	skipUnlessPosix(t)
	src := t.TempDir()
	write(t, filepath.Join(src, "real"), "x")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, err := SaveTree(staging, []string{src, filepath.Join(src, "absent")}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 1, size)
}

func TestSaveReportsTheTotalSize(t *testing.T) {
	skipUnlessPosix(t)
	src := t.TempDir()
	write(t, filepath.Join(src, "a"), "12345")
	write(t, filepath.Join(src, "b"), "678")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, err := SaveTree(staging, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 8, size)
}

// The container root is refused the way the archive form refused it, so a cache cannot
// be declared over the whole filesystem.
func TestSaveRefusesTheRoot(t *testing.T) {
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, err := SaveTree(staging, []string{"/"}, CopyLimits{})

	require.NoError(t, err)
	assert.Zero(t, size)
}

// skipUnlessPosix skips a test that depends on mirroring absolute container paths.
//
// An entry stores a cached path by its absolute name with the leading separator
// dropped, which a Windows drive letter cannot be. The agent runs only in Linux
// containers, so this is the platform the behaviour is defined on rather than a gap to
// paper over.
func skipUnlessPosix(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("cached paths are absolute POSIX paths; the agent runs in Linux containers")
	}
}
