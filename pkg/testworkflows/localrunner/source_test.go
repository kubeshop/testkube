package localrunner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteSourceArchiveAppliesDefaultsAndExplicitOverrides(t *testing.T) {
	root := t.TempDir()
	writeSourceFileForTest(t, root, "keep.txt", "keep")
	writeSourceFileForTest(t, root, ".git/config", "git")
	writeSourceFileForTest(t, root, ".testkube/cache", "cache")
	writeSourceFileForTest(t, root, "node_modules/ignored.js", "ignored")
	writeSourceFileForTest(t, root, "node_modules/needed.js", "needed")
	writeSourceFileForTest(t, root, "vendor/dependency.go", "vendor")
	writeSourceFileForTest(t, root, "build/output", "build")
	writeSourceFileForTest(t, root, "nested/dist/output", "dist")
	writeSourceFileForTest(t, root, ".idea/workspace.xml", "idea")
	writeSourceFileForTest(t, root, ".DS_Store", "os")
	writeSourceFileForTest(t, root, "scratch.swp", "editor")

	archive := &bytes.Buffer{}
	summary, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		Includes:  []string{"node_modules/needed.js"},
		MaxBytes:  1024,
	}, archive)
	require.NoError(t, err)
	assert.Equal(t, SourceSummary{Bytes: int64(len("keep") + len("needed")), Files: 2}, summary)

	entries := readSourceArchive(t, archive.Bytes())
	assert.Contains(t, entries, "keep.txt")
	assert.Contains(t, entries, "node_modules/needed.js")
	assert.NotContains(t, entries, ".git/config")
	assert.NotContains(t, entries, ".testkube/cache")
	assert.NotContains(t, entries, "node_modules/ignored.js")
	assert.NotContains(t, entries, "vendor/dependency.go")
	assert.NotContains(t, entries, "build/output")
	assert.NotContains(t, entries, "nested/dist/output")
	assert.NotContains(t, entries, ".idea/workspace.xml")
	assert.NotContains(t, entries, ".DS_Store")
	assert.NotContains(t, entries, "scratch.swp")
}

func TestWriteSourceArchiveExplicitExcludeWinsOverEarlierIgnoreRules(t *testing.T) {
	root := t.TempDir()
	writeSourceFileForTest(t, root, ".testkubeignore", "!generated/keep.txt\n")
	writeSourceFileForTest(t, root, "generated/keep.txt", "keep")
	writeSourceFileForTest(t, root, "generated/remove.txt", "remove")
	writeSourceFileForTest(t, root, "other.txt", "other")

	archive := &bytes.Buffer{}
	_, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		Excludes:  []string{"generated/**"},
		MaxBytes:  1024,
	}, archive)
	require.NoError(t, err)

	entries := readSourceArchive(t, archive.Bytes())
	assert.Contains(t, entries, ".testkubeignore")
	assert.Contains(t, entries, "other.txt")
	assert.NotContains(t, entries, "generated/keep.txt")
	assert.NotContains(t, entries, "generated/remove.txt")
}

func TestWriteSourceArchiveHonorsOrderedTestkubeIgnore(t *testing.T) {
	root := t.TempDir()
	writeSourceFileForTest(t, root, ".testkubeignore", "ignored/\n!ignored/keep.txt\n")
	writeSourceFileForTest(t, root, "ignored/drop.txt", "drop")
	writeSourceFileForTest(t, root, "ignored/keep.txt", "keep")
	writeSourceFileForTest(t, root, "visible.txt", "visible")

	archive := &bytes.Buffer{}
	_, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		MaxBytes:  1024,
	}, archive)
	require.NoError(t, err)

	entries := readSourceArchive(t, archive.Bytes())
	assert.Contains(t, entries, ".testkubeignore")
	assert.Contains(t, entries, "ignored/keep.txt")
	assert.NotContains(t, entries, "ignored/drop.txt")
	assert.Contains(t, entries, "visible.txt")
}

func TestWriteSourceArchivePreservesSafeRelativeSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are not portable on Windows test hosts")
	}

	root := t.TempDir()
	writeSourceFileForTest(t, root, "target.txt", "target")
	require.NoError(t, os.Symlink("target.txt", filepath.Join(root, "alias.txt")))

	archive := &bytes.Buffer{}
	summary, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		MaxBytes:  1024,
	}, archive)
	require.NoError(t, err)
	assert.Equal(t, SourceSummary{Bytes: int64(len("target")), Files: 2}, summary)

	entries := readSourceArchive(t, archive.Bytes())
	header, ok := entries["alias.txt"]
	require.True(t, ok)
	assert.Equal(t, byte(tar.TypeSymlink), header.Typeflag)
	assert.Equal(t, "target.txt", header.Linkname)
}

func TestWriteSourceArchiveRejectsUnsafeSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are not portable on Windows test hosts")
	}

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "absolute", target: "/etc/passwd", want: "absolute symlink target"},
		{name: "parent traversing", target: "../outside", want: "parent-traversing symlink target"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.Symlink(test.target, filepath.Join(root, "unsafe-link")))

			_, err := WriteSourceArchive(context.Background(), SourceOptions{
				Directory: root,
				MaxBytes:  1024,
			}, io.Discard)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestWriteSourceArchiveRejectsOversizedAndInvalidInput(t *testing.T) {
	root := t.TempDir()
	writeSourceFileForTest(t, root, "large.txt", "0123456789")

	tests := []struct {
		name string
		opts SourceOptions
		want string
	}{
		{
			name: "max bytes must be positive",
			opts: SourceOptions{Directory: root, MaxBytes: 0},
			want: "max source bytes must be greater than zero",
		},
		{
			name: "file is larger than the limit",
			opts: SourceOptions{Directory: root, MaxBytes: 9},
			want: "source archive exceeds max source bytes",
		},
		{
			name: "unsafe flag pattern",
			opts: SourceOptions{Directory: root, Excludes: []string{"../outside"}, MaxBytes: 1024},
			want: "source exclude pattern",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := WriteSourceArchive(context.Background(), test.opts, io.Discard)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestWriteSourceArchiveEmptyAndCancelled(t *testing.T) {
	root := t.TempDir()
	archive := &bytes.Buffer{}
	summary, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		MaxBytes:  1024,
	}, archive)
	require.NoError(t, err)
	assert.Equal(t, SourceSummary{}, summary)
	assert.Empty(t, readSourceArchive(t, archive.Bytes()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = WriteSourceArchive(ctx, SourceOptions{Directory: root, MaxBytes: 1024}, io.Discard)
	require.ErrorIs(t, err, context.Canceled)

	writeSourceFileForTest(t, root, "large-enough-to-stream.txt", string(nonRepeatingBytes(256*1024)))
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err = WriteSourceArchive(ctx, SourceOptions{Directory: root, MaxBytes: 512 * 1024}, writerFunc(func(data []byte) (int, error) {
		cancel()
		return len(data), nil
	}))
	require.ErrorIs(t, err, context.Canceled)
}

func TestWriteSourceArchiveFailsWhenAFileChangesDuringPackaging(t *testing.T) {
	root := t.TempDir()
	contents := nonRepeatingBytes(2 * 1024 * 1024)
	writeSourceFileForTest(t, root, "changing.txt", string(contents))
	filePath := filepath.Join(root, "changing.txt")

	mutated := false
	_, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		MaxBytes:  3 * 1024 * 1024,
	}, writerFunc(func(data []byte) (int, error) {
		if !mutated {
			mutated = true
			require.NoError(t, os.WriteFile(filePath, append(contents, 'x'), 0o644))
		}
		return len(data), nil
	}))
	require.True(t, mutated, "the compressed archive should stream while reading a multi-megabyte file")
	require.Error(t, err)
}

func TestWriteSourceArchiveFailsForUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission checks are not meaningful on this test host")
	}

	root := t.TempDir()
	filePath := filepath.Join(root, "unreadable.txt")
	writeSourceFileForTest(t, root, "unreadable.txt", "secret")
	require.NoError(t, os.Chmod(filePath, 0))
	t.Cleanup(func() { _ = os.Chmod(filePath, 0o644) })

	_, err := WriteSourceArchive(context.Background(), SourceOptions{
		Directory: root,
		MaxBytes:  1024,
	}, io.Discard)
	require.Error(t, err)
}

func TestSafeArchiveNameRejectsAbsoluteAndEscapingNames(t *testing.T) {
	assert.True(t, safeArchiveName("nested/file.txt"))
	assert.False(t, safeArchiveName("/etc/passwd"))
	assert.False(t, safeArchiveName("../outside"))
	assert.False(t, safeArchiveName("nested/../outside"))
}

func writeSourceFileForTest(t *testing.T, root, name, contents string) {
	t.Helper()
	filePath := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
	require.NoError(t, os.WriteFile(filePath, []byte(contents), 0o644))
}

func readSourceArchive(t *testing.T, compressed []byte) map[string]*tar.Header {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	entries := map[string]*tar.Header{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		entries[header.Name] = header
	}
	return entries
}

type writerFunc func([]byte) (int, error)

func (fn writerFunc) Write(data []byte) (int, error) {
	return fn(data)
}

func nonRepeatingBytes(size int) []byte {
	data := make([]byte, size)
	state := uint32(0x9e3779b9)
	for index := range data {
		state = state*1664525 + 1013904223
		data[index] = byte(state >> 24)
	}
	return data
}
