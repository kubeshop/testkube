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
