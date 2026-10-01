package volume

import (
	"os"
	"path/filepath"
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
	_, _, err := SaveTree(staging, paths, CopyLimits{})
	require.NoError(t, err)
	root, err := os.OpenRoot(staging)
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })
	return root
}

func TestSaveAndRestoreRoundTripATree(t *testing.T) {
	src := posixDir(t)
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
	declared := posixDir(t)
	smuggled := posixDir(t)
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
	src := posixDir(t)
	write(t, filepath.Join(src, "big"), "0123456789")
	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{MaxTotalBytes: 4})

	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestRestoreStopsAtTheEntryLimit(t *testing.T) {
	src := posixDir(t)
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
	src := posixDir(t)
	write(t, filepath.Join(src, "real"), "x")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{src, filepath.Join(src, "absent")}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 1, size)
}

func TestSaveReportsTheTotalSize(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "a"), "12345")
	write(t, filepath.Join(src, "b"), "678")
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{src}, CopyLimits{})

	require.NoError(t, err)
	assert.EqualValues(t, 8, size)
}

// The container root is refused the way the archive form refused it, so a cache cannot
// be declared over the whole filesystem.
func TestSaveRefusesTheRoot(t *testing.T) {
	staging := filepath.Join(t.TempDir(), EntryRoot)
	require.NoError(t, os.MkdirAll(staging, 0o777))

	size, _, err := SaveTree(staging, []string{"/"}, CopyLimits{})

	require.NoError(t, err)
	assert.Zero(t, size)
}

// A symlink already sitting inside a declared path is attacker-controlled input: the
// entry may have been written by another workflow under an environment-scoped key. A
// restore that wrote by absolute name would follow it and overwrite a file the step
// never declared, which is what writing through an os.Root on the declared path stops.
func TestRestoreDoesNotFollowASymlinkOutOfTheDeclaredPath(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside")
	require.NoError(t, os.WriteFile(outside, []byte("untouched"), 0o666))

	// The entry carries a plain file at <declared>/escape.
	declared := posixDir(t)
	write(t, filepath.Join(declared, "escape"), "from the entry")
	entry := entryFrom(t, []string{declared})
	require.NoError(t, os.RemoveAll(declared))

	// The destination already holds a symlink at that name, pointing outside.
	require.NoError(t, os.MkdirAll(declared, 0o777))
	require.NoError(t, os.Symlink(outside, filepath.Join(declared, "escape")))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})
	require.NoError(t, err)

	body, readErr := os.ReadFile(outside)
	require.NoError(t, readErr)
	assert.Equal(t, "untouched", string(body),
		"a restore must not write through a symlink that leaves the declared path")
}

// node_modules/.bin is entirely symlinks. Dropping them would restore an incomplete
// tree that still answers as an exact hit, so the step would never repair it.
func TestSaveAndRestorePreserveSymlinks(t *testing.T) {
	src := posixDir(t)
	write(t, filepath.Join(src, "pkg", "cli.js"), "#!/usr/bin/env node")
	require.NoError(t, os.MkdirAll(filepath.Join(src, ".bin"), 0o777))
	require.NoError(t, os.Symlink("../pkg/cli.js", filepath.Join(src, ".bin", "cli")))

	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(src))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{})
	require.NoError(t, err)

	link, readErr := os.Readlink(filepath.Join(src, ".bin", "cli"))
	require.NoError(t, readErr, "the link must come back as a link")
	assert.Equal(t, "../pkg/cli.js", filepath.ToSlash(link),
		"a relative link must stay relative, or it would name the pod that saved it")

	// And it still resolves, which is the whole point of carrying it.
	body, readErr := os.ReadFile(filepath.Join(src, ".bin", "cli"))
	require.NoError(t, readErr)
	assert.Equal(t, "#!/usr/bin/env node", string(body))
}

// Restoring over a tree that is already there is ordinary: a step may declare a path
// its checkout populated.
func TestRestoreOverwritesAnExistingFile(t *testing.T) {
	declared := posixDir(t)
	write(t, filepath.Join(declared, "dep"), "from the entry")
	entry := entryFrom(t, []string{declared})

	require.NoError(t, os.WriteFile(filepath.Join(declared, "dep"), []byte("stale"), 0o666))

	_, err := RestoreTree(entry, []string{declared}, CopyLimits{})
	require.NoError(t, err)

	body, readErr := os.ReadFile(filepath.Join(declared, "dep"))
	require.NoError(t, readErr)
	assert.Equal(t, "from the entry", string(body))
}

// posixDir returns a temp directory named the way a cached path is: an absolute POSIX
// path. On Windows the drive letter is dropped, which still resolves to the same
// directory on the current drive, so the mirroring these tests exercise behaves as it
// does in the Linux containers the agent actually runs in.
func posixDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.ToSlash(dir[len(filepath.VolumeName(dir)):])
}
