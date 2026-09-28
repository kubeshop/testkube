package libs

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestFsLibGlob(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	_ = afero.WriteFile(fsys.Fs, "etc/file1.txt", nil, 0644)
	_ = afero.WriteFile(fsys.Fs, "else/file1.txt", nil, 0644)
	_ = afero.WriteFile(fsys.Fs, "another-file.txt", nil, 0644)
	_ = afero.WriteFile(fsys.Fs, "etc/nested/file2.json", nil, 0644)
	machine := NewFsMachine(fsys, "/etc")
	assert.Equal(t, []string{"/etc/file1.txt", "/etc/nested/file2.json"}, expressions.MustCall(machine, "glob", "**/*"))
	assert.Equal(t, []string{"/etc/file1.txt"}, expressions.MustCall(machine, "glob", "*"))
	assert.Equal(t, []string{"/etc/nested/file2.json"}, expressions.MustCall(machine, "glob", "**/*.json"))
	assert.Equal(t, []string{"/etc/file1.txt", "/etc/nested/file2.json"}, expressions.MustCall(machine, "glob", "**/*.json", "*.txt"))
	assert.Equal(t, []string{"/another-file.txt", "/else/file1.txt", "/etc/file1.txt"}, expressions.MustCall(machine, "glob", "/**/*.txt"))
	assert.Equal(t, []string{"/another-file.txt", "/etc/file1.txt"}, expressions.MustCall(machine, "glob", "/**/*.txt", "!/else/**/*"))
}

func TestFsLibHashFiles(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	_ = afero.WriteFile(fsys.Fs, "etc/a.lock", []byte("one"), 0644)
	_ = afero.WriteFile(fsys.Fs, "etc/b.lock", []byte("two"), 0644)
	machine := NewFsMachine(fsys, "/etc")

	digest, ok := expressions.MustCall(machine, "hash_files", "*.lock").(string)
	assert.True(t, ok)
	assert.Len(t, digest, 64)

	// Stable across calls, and independent of the order the patterns matched in -
	// otherwise the same dependency tree would miss its own cache entry.
	assert.Equal(t, digest, expressions.MustCall(machine, "hash_files", "*.lock"))
	assert.Equal(t, digest, expressions.MustCall(machine, "hash_files", "b.lock", "a.lock"))

	// Contents are what matter: this is the whole reason hash_files exists.
	_ = afero.WriteFile(fsys.Fs, "etc/a.lock", []byte("changed"), 0644)
	assert.NotEqual(t, digest, expressions.MustCall(machine, "hash_files", "*.lock"))

	// Adding a file changes the digest too.
	before := expressions.MustCall(machine, "hash_files", "*.lock")
	_ = afero.WriteFile(fsys.Fs, "etc/c.lock", []byte("three"), 0644)
	assert.NotEqual(t, before, expressions.MustCall(machine, "hash_files", "*.lock"))

	// The ignore patterns come from glob() for free.
	assert.Equal(t,
		expressions.MustCall(machine, "hash_files", "a.lock"),
		expressions.MustCall(machine, "hash_files", "*.lock", "!b.lock", "!c.lock"))

	// No match is empty, not an error: a lockfile that does not exist yet is normal
	// while a workflow is being written. Callers treat "" as "do not cache".
	assert.Equal(t, "", expressions.MustCall(machine, "hash_files", "*.nope"))

	// The same has to hold for a literal path, which is the form the documentation
	// shows and the form a real key uses - hash_files("package-lock.json"), not a
	// wildcard. The two reach the empty answer by different routes: a wildcard's search
	// root is a directory that exists, so the walk runs and matches nothing, while a
	// literal's root is the missing path itself, so the walk fails before it can match
	// anything. What makes the answers agree is the walk callback discarding that error
	// - one `err != nil` whose loss nothing else would notice, and which would turn the
	// documented "do not cache" into a failed step.
	assert.Equal(t, "", expressions.MustCall(machine, "hash_files", "package-lock.json"))
	assert.Equal(t, "", expressions.MustCall(machine, "hash_files", "./package-lock.json"))
	assert.Equal(t, "", expressions.MustCall(machine, "hash_files", "/etc/package-lock.json"))
	// And when whole directories along the way are absent, not only the file.
	assert.Equal(t, "", expressions.MustCall(machine, "hash_files", "nested/deep/package-lock.json"))

	// A missing literal beside a present one is not fatal either: the key still derives
	// from what is there, rather than the step losing its cache outright.
	assert.Equal(t,
		expressions.MustCall(machine, "hash_files", "a.lock"),
		expressions.MustCall(machine, "hash_files", "a.lock", "package-lock.json"))
}

// TestFsLibHashFilesDiffersFromGlob documents the footgun hash_files exists to avoid:
// hash(glob(...)) digests the list of paths, so editing a lockfile leaves it unchanged.
func TestFsLibHashFilesDiffersFromGlob(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	_ = afero.WriteFile(fsys.Fs, "etc/a.lock", []byte("one"), 0644)
	machine := NewFsMachine(fsys, "/etc")

	pathsBefore := expressions.MustCall(machine, "glob", "*.lock")
	hashBefore := expressions.MustCall(machine, "hash_files", "*.lock")

	_ = afero.WriteFile(fsys.Fs, "etc/a.lock", []byte("two"), 0644)

	assert.Equal(t, pathsBefore, expressions.MustCall(machine, "glob", "*.lock"))
	assert.NotEqual(t, hashBefore, expressions.MustCall(machine, "hash_files", "*.lock"))
}

func TestFsLibRead(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	_ = afero.WriteFile(fsys.Fs, "etc/file1.txt", []byte("foo"), 0644)
	_ = afero.WriteFile(fsys.Fs, "another-file.txt", []byte("bar"), 0644)
	machine := NewFsMachine(fsys, "/etc")
	assert.Equal(t, "foo", expressions.MustCall(machine, "file", "file1.txt"))
	assert.Equal(t, "foo", expressions.MustCall(machine, "file", "/etc/file1.txt"))
	assert.Equal(t, "bar", expressions.MustCall(machine, "file", "../another-file.txt"))
	assert.Equal(t, "bar", expressions.MustCall(machine, "file", "/another-file.txt"))
}

// erroringFS answers one subtree with an I/O failure and serves the rest normally.
type erroringFS struct {
	fs.FS
	failAt string
	err    error
}

func (f erroringFS) Open(name string) (fs.File, error) {
	if name == f.failAt {
		return nil, f.err
	}
	return f.FS.Open(name)
}

func (f erroringFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.failAt {
		return nil, f.err
	}
	return fs.ReadDir(f.FS, name)
}

// TestFsLibSurfacesTraversalErrors separates the two things a walk failure can mean.
//
// The callback discarded every error it was handed, which made "this path is not there"
// and "this directory cannot be read" indistinguishable. The first is the documented
// empty match that tells a caller not to cache. The second silently narrowed the match
// set, so hash_files returned a digest over the files it happened to manage to read - a
// key that looks entirely valid while naming a different dependency set than it claims,
// and that every later run then restores from.
func TestFsLibSurfacesTraversalErrors(t *testing.T) {
	// Keyed under whatever absPath resolves the working directory to, so the walk
	// reaches the fixture on every OS: on Windows filepath.Abs prefixes the drive,
	// which is why the rest of this package cannot match anything here.
	root := strings.TrimPrefix(absPath("", "/work"), "/")
	base := fstest.MapFS{
		root + "/a.lock":        &fstest.MapFile{Data: []byte("one")},
		root + "/deep/b.lock":   &fstest.MapFile{Data: []byte("two")},
		root + "/deep/c/d.lock": &fstest.MapFile{Data: []byte("three")},
	}

	call := func(machine expressions.Machine, name, arg string) error {
		_, _, err := machine.Call(name, []expressions.CallArgument{{Expression: expressions.NewValue(arg)}})
		return err
	}

	denied := errors.New("permission denied reading the directory")
	failing := erroringFS{FS: base, failAt: root + "/deep", err: denied}
	machine := NewFsMachine(failing, "/work")

	err := call(machine, "glob", "**")
	require.Error(t, err, "a directory that cannot be read must not pass for an empty one")
	assert.ErrorContains(t, err, "permission denied")

	err = call(machine, "hash_files", "**")
	require.Error(t, err, "a partial digest is worse than none: it looks like a valid key")
	assert.ErrorContains(t, err, "permission denied")

	// The readable filesystem still answers, so the test is not simply erroring on
	// everything.
	require.NoError(t, call(NewFsMachine(base, "/work"), "hash_files", "**"))
}
