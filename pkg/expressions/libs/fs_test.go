package libs

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

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
