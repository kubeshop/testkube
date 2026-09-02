package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildMixedTarball(t *testing.T, regContent string) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}))

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/file.txt",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len(regContent)),
	}))
	_, err := tw.Write([]byte(regContent))
	require.NoError(t, err)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/link.txt",
		Typeflag: tar.TypeSymlink,
		Linkname: "file.txt",
		Mode:     0o777,
	}))

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf
}

func TestUnpackTarball_ExtractIsIdempotentOnRetry(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "hello")))
	assert.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "hello")))

	data, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	target, err := os.Readlink(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "file.txt", target)
}

func TestUnpackTarball_ExtractOverwritesOnRetry(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "first")))

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "sub/", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "sub/other.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 5,
	}))
	_, err := tw.Write([]byte("other"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "sub/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 6}))
	_, err = tw.Write([]byte("second"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "sub/link.txt", Typeflag: tar.TypeSymlink, Linkname: "other.txt", Mode: 0o777,
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	require.NoError(t, UnpackTarball(dir, buf))

	got, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "second", string(got))

	target, err := os.Readlink(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "other.txt", target)
}

// tarballOf builds a gzipped tar from the given headers, writing body for regular
// files, so that each hostile-archive case reads as the archive it describes.
func tarballOf(t *testing.T, entries ...tar.Header) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	for i := range entries {
		header := entries[i]
		body := make([]byte, header.Size)
		require.NoError(t, tw.WriteHeader(&header))
		if header.Typeflag == tar.TypeReg {
			_, err := tw.Write(body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf
}

// TestUnpackTarball_SymlinkDirWriteThroughIsRefused is the important one.
//
// Neither entry's *name* is unsafe: "link" and "link/escaped.txt" contain no ".." and
// no leading slash, so the header checks pass both. The escape comes from resolving the
// second through the first. Before extraction went through os.Root this wrote outside
// the destination entirely, which is arbitrary file write from any archive - and a
// dependency cache is an archive an earlier, possibly different, workflow wrote.
func TestUnpackTarball_SymlinkDirWriteThroughIsRefused(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	err := UnpackTarball(dir, tarballOf(t,
		tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: outside, Mode: 0o777},
		tar.Header{Name: "link/escaped.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 5},
	))

	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(outside, "escaped.txt"))
	assert.True(t, os.IsNotExist(statErr), "the write escaped the destination directory")
}

func TestUnpackTarball_RejectsUnsafeEntryNames(t *testing.T) {
	for _, name := range []string{"../escape.txt", "/abs.txt", "sub/../../escape.txt", ".."} {
		t.Run(name, func(t *testing.T) {
			err := UnpackTarball(t.TempDir(), tarballOf(t,
				tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
			))
			assert.ErrorContains(t, err, "unsafe file path")
		})
	}
}

// TestUnpackTarball_KeepsRelativeSymlinks guards the feature's usefulness: node_modules
// under pnpm and yarn workspaces is built out of symlinks, so rejecting them outright
// would be a safe but useless unpacker.
func TestUnpackTarball_KeepsRelativeSymlinks(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, tarballOf(t,
		tar.Header{Name: "pkg", Typeflag: tar.TypeDir, Mode: 0o755},
		tar.Header{Name: "pkg/real.js", Typeflag: tar.TypeReg, Mode: 0o644, Size: 3},
		tar.Header{Name: "bin", Typeflag: tar.TypeDir, Mode: 0o755},
		tar.Header{Name: "bin/cli", Typeflag: tar.TypeSymlink, Linkname: "../pkg/real.js", Mode: 0o777},
	)))

	target, err := os.Readlink(filepath.Join(dir, "bin", "cli"))
	require.NoError(t, err)
	assert.Equal(t, "../pkg/real.js", target)
	// Reachable through the link, which is the point of keeping it.
	_, err = os.Stat(filepath.Join(dir, "bin", "cli"))
	assert.NoError(t, err)
}

func TestUnpackTarball_DropsSetuidBit(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, tarballOf(t,
		tar.Header{Name: "payload", Typeflag: tar.TypeReg, Mode: 0o4755, Size: 2},
	)))

	stat, err := os.Stat(filepath.Join(dir, "payload"))
	require.NoError(t, err)
	assert.Zero(t, stat.Mode()&os.ModeSetuid, "setuid bit survived extraction")
	assert.Zero(t, stat.Mode()&os.ModeSetgid, "setgid bit survived extraction")
}

func TestUnpackTarball_EnforcesLimits(t *testing.T) {
	t.Run("total bytes", func(t *testing.T) {
		err := UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "big.bin", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4096},
		), WithMaxTotalBytes(1024))
		assert.ErrorContains(t, err, "expands to more than")
	})

	t.Run("total bytes across entries", func(t *testing.T) {
		err := UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "a.bin", Typeflag: tar.TypeReg, Mode: 0o644, Size: 600},
			tar.Header{Name: "b.bin", Typeflag: tar.TypeReg, Mode: 0o644, Size: 600},
		), WithMaxTotalBytes(1024))
		assert.ErrorContains(t, err, "expands to more than")
	})

	t.Run("entry count", func(t *testing.T) {
		err := UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "a", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
			tar.Header{Name: "b", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
			tar.Header{Name: "c", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		), WithMaxEntries(2))
		assert.ErrorContains(t, err, "more than 2 entries")
	})

	t.Run("within limits", func(t *testing.T) {
		assert.NoError(t, UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "a.bin", Typeflag: tar.TypeReg, Mode: 0o644, Size: 600},
		), WithMaxTotalBytes(1024), WithMaxEntries(10)))
	})

	t.Run("unlimited by default", func(t *testing.T) {
		assert.NoError(t, UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "big.bin", Typeflag: tar.TypeReg, Mode: 0o644, Size: 8192},
		)))
	})
}

func TestUnpackTarball_RejectsHardLinks(t *testing.T) {
	err := UnpackTarball(t.TempDir(), tarballOf(t,
		tar.Header{Name: "a", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		tar.Header{Name: "b", Typeflag: tar.TypeLink, Linkname: "a", Mode: 0o644},
	))
	assert.ErrorContains(t, err, "unknown entry type")
}

func TestUnpackTarball_FreshExtract(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "payload")))

	data, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))

	_, err = os.Lstat(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
}
