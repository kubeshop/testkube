//go:build unix

package volume

import (
	"io/fs"
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

// "..cache" is an ordinary directory name, not an escape. Classifying it as one by its
// first two characters left the chain above it at whatever the step's umask gave -
// which is the mode the agent cannot sweep through, so that subtree would never be
// reclaimed from a volume the whole cluster shares.
func TestSaveSetsTheModeUnderADirectoryNamedLikeADotDot(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	src := posixDir(t) + "/..cache/deps"
	write(t, filepath.FromSlash(src+"/dep"), "x")

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

// A file on the volume is read by an execution that may run as another user, so what is
// stored is widened to be readable by anyone - which loses the source's own mode. The
// archive backend has no such problem, carrying each mode in a tar header, so the same
// workflow behaved differently depending on which backend held its cache: a key saved
// 0600 came back 0666, and ssh refuses a key that anyone could read.
//
// Unix-only because the assertion is about POSIX permission bits, which is not what
// Windows enforces.
func TestSaveAndRestorePreserveModes(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	src := posixDir(t)
	write(t, filepath.FromSlash(src+"/pkg/index.js"), "ordinary")
	write(t, filepath.FromSlash(src+"/.npmrc"), "//registry:_authToken=secret")
	write(t, filepath.FromSlash(src+"/bin/tool"), "#!/bin/sh")
	require.NoError(t, os.MkdirAll(filepath.FromSlash(src+"/private"), 0o755))
	write(t, filepath.FromSlash(src+"/private/key"), "-----BEGIN-----")

	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/.npmrc"), 0o600))
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/bin/tool"), 0o755))
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/private/key"), 0o400))
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/private"), 0o700))

	entry := entryFrom(t, []string{src})
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/private"), 0o755))
	require.NoError(t, os.RemoveAll(filepath.FromSlash(src)))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{})
	require.NoError(t, err)

	for name, want := range map[string]fs.FileMode{
		"/pkg/index.js": 0o644,
		"/.npmrc":       0o600,
		"/bin/tool":     0o755,
		"/private":      0o700,
		"/private/key":  0o400,
	} {
		info, statErr := os.Lstat(filepath.FromSlash(src + name))
		require.NoError(t, statErr, name)
		assert.Equal(t, want, info.Mode().Perm(), "%s came back with the wrong permissions", name)
	}
}

// The declared directory has a recorded mode like anything else in the entry - SaveTree
// walks it first - and it is restored through the root opened on it, which it addresses
// as ".". Skipping it left a directory saved 0700 at whatever the umask gave a fresh
// one, or at whatever mode a previous run had left behind.
func TestRestoreSetsTheDeclaredDirectorysOwnMode(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	src := posixDir(t) + "/private"
	write(t, filepath.FromSlash(src+"/key"), "-----BEGIN-----")
	require.NoError(t, os.Chmod(filepath.FromSlash(src), 0o700))

	entry := entryFrom(t, []string{src})
	require.NoError(t, os.Chmod(filepath.FromSlash(src), 0o755))
	require.NoError(t, os.RemoveAll(filepath.FromSlash(src)))

	_, err := RestoreTree(entry, []string{src}, CopyLimits{})
	require.NoError(t, err)

	info, statErr := os.Lstat(filepath.FromSlash(src))
	require.NoError(t, statErr)
	assert.Equal(t, fs.FileMode(0o700), info.Mode().Perm(),
		"the declared directory's own permissions are part of what was cached")
}

// The manifest records only what differs from the defaults, so the defaults have to be
// applied rather than assumed. MkdirAll and OpenFile take their mode through the
// process umask, and a restoring container is entitled to any umask it likes: under
// 0077 a tree cached at an ordinary 0755/0644 came back 0700/0600, private to whoever
// restored it, where the archive backend sets every mode from its tar header.
//
// Saved under a permissive umask and restored under a strict one, which is the shape of
// the problem: two different containers, each with its own.
func TestRestoreAppliesTheDefaultsUnderAStrictUmask(t *testing.T) {
	saved := syscall.Umask(0o022)
	src := posixDir(t) + "/pkg"
	write(t, filepath.FromSlash(src+"/nested/index.js"), "ordinary")
	require.NoError(t, os.Chmod(filepath.FromSlash(src), 0o755))
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/nested"), 0o755))
	require.NoError(t, os.Chmod(filepath.FromSlash(src+"/nested/index.js"), 0o644))

	entry := entryFrom(t, []string{src})
	require.NoError(t, os.RemoveAll(filepath.FromSlash(src)))
	syscall.Umask(saved)

	// Nothing differed from the defaults, so the entry records nothing at all - which
	// is exactly the case the defaults have to carry on their own.
	_, statErr := os.Stat(filepath.Join(entry.Name(), ModesName))
	require.True(t, os.IsNotExist(statErr), "this tree is all defaults")

	strict := syscall.Umask(0o077)
	defer syscall.Umask(strict)

	_, err := RestoreTree(entry, []string{src}, CopyLimits{})
	require.NoError(t, err)

	for name, want := range map[string]fs.FileMode{
		"":                 0o755,
		"/nested":          0o755,
		"/nested/index.js": 0o644,
	} {
		info, statErr := os.Lstat(filepath.FromSlash(src + name))
		require.NoError(t, statErr, name)
		assert.Equal(t, want, info.Mode().Perm(), "%s came back at the restoring umask", name)
	}
}

// A recorded mode can take a directory's write bit away, and a later declared path can
// still fail. Setting modes as each path finished left the caller unable to clear what
// had been written - the tree it has to empty is one the restore had just made
// unwritable - so a reported miss left part of a cache behind for the install to find.
//
// Here the first path is cached 0500 and the second is refused by the entry limit.
func TestRestoreLeavesTheFirstPathWritableWhenALaterOneFails(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	first, second := twoPathsOneLimitApart(t)
	require.NoError(t, os.Chmod(filepath.FromSlash(first), 0o500))

	entry := entryFrom(t, []string{first, second})
	require.NoError(t, os.Chmod(filepath.FromSlash(first), 0o755))
	require.NoError(t, os.RemoveAll(filepath.FromSlash(first)))
	require.NoError(t, os.RemoveAll(filepath.FromSlash(second)))

	// Enough for the first path and not for the second, so the restore fails part way.
	_, err := RestoreTree(entry, []string{first, second}, CopyLimits{MaxEntries: entriesForFirstPathOnly})
	require.Error(t, err)

	// What the caller does next: empty the declared paths. It cannot, if the restore
	// has already taken the write bit off what it wrote.
	info, statErr := os.Lstat(filepath.FromSlash(first))
	require.NoError(t, statErr)
	assert.NotZero(t, info.Mode().Perm()&0o200,
		"a failed restore must leave behind nothing its caller cannot clear")
}
