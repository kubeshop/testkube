package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path"
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

// TestUnpackTarball_AllowedRootsConfineACacheArchive guards the case that makes an
// environment-scoped cache safe to read.
//
// A cache is unpacked at "/", because its entries name paths across several volumes, so
// os.Root bounds the extraction to the whole filesystem - which bounds nothing. The
// paths the step asked to restore are what actually confine it, so an archive reaching
// beyond them is refused rather than partly written.
//
// The roots here are expressed relative to a temporary destination so the assertions
// describe the check itself rather than the host's idea of an absolute path.
func TestUnpackTarball_AllowedRootsConfineACacheArchive(t *testing.T) {
	rootIn := func(dir, name string) string {
		return path.Join(path.Clean(filepath.ToSlash(dir)), name)
	}

	t.Run("an entry outside the declared paths is refused", func(t *testing.T) {
		dir := t.TempDir()

		err := UnpackTarball(dir, tarballOf(t,
			tar.Header{Name: "elsewhere/planted.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 5},
		), WithAllowedRoots(rootIn(dir, "deps")))

		require.ErrorContains(t, err, "outside the paths this step asked to restore")
		_, statErr := os.Stat(filepath.Join(dir, "elsewhere", "planted.txt"))
		assert.True(t, os.IsNotExist(statErr), "a refused entry must not be written")
	})

	t.Run("entries inside the declared paths are written", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, UnpackTarball(dir, tarballOf(t,
			tar.Header{Name: "deps/pkg/index.js", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4},
		), WithAllowedRoots(rootIn(dir, "deps"))))

		_, err := os.Stat(filepath.Join(dir, "deps", "pkg", "index.js"))
		assert.NoError(t, err)
	})

	t.Run("a sibling sharing a prefix is not allowed", func(t *testing.T) {
		// "deps-evil" must not pass a check for "deps".
		dir := t.TempDir()

		err := UnpackTarball(dir, tarballOf(t,
			tar.Header{Name: "deps-evil/planted.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		), WithAllowedRoots(rootIn(dir, "deps")))

		assert.ErrorContains(t, err, "outside the paths this step asked to restore")
	})

	t.Run("several declared paths are all allowed", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, UnpackTarball(dir, tarballOf(t,
			tar.Header{Name: "node_modules/a.js", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
			tar.Header{Name: "m2/b.jar", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		), WithAllowedRoots(rootIn(dir, "node_modules"), rootIn(dir, "m2"))))

		_, err := os.Stat(filepath.Join(dir, "node_modules", "a.js"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, "m2", "b.jar"))
		assert.NoError(t, err)
	})

	t.Run("no roots leaves the transfer path unrestricted", func(t *testing.T) {
		// The pod-to-pod transfer archive comes from a sibling container in the same
		// execution, so it is the caller's own doing and needs no allowlist.
		assert.NoError(t, UnpackTarball(t.TempDir(), tarballOf(t,
			tar.Header{Name: "anywhere/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		)))
	})

	// The other half: an archive cannot smuggle a write past the allowlist by planting a
	// symlink inside an allowed path and writing through it. The link's target is not
	// what is checked - os.Root refuses the traversal.
	t.Run("a symlink inside an allowed path is not a way out", func(t *testing.T) {
		dir := t.TempDir()
		outside := t.TempDir()

		err := UnpackTarball(dir, tarballOf(t,
			tar.Header{Name: "deps/link", Typeflag: tar.TypeSymlink, Linkname: outside, Mode: 0o777},
			tar.Header{Name: "deps/link/planted.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		), WithAllowedRoots(rootIn(dir, "deps")))

		require.Error(t, err)
		_, statErr := os.Stat(filepath.Join(outside, "planted.txt"))
		assert.True(t, os.IsNotExist(statErr), "the write escaped through a symlink inside an allowed path")
	})
}

// TestWriteTarballFrom_RoundTripsThroughTheAllowlist is the regression guard on the
// allowlist: an archive this code actually produces for a cache must still restore in
// full. If the walker emitted anything outside the packed paths - a parent directory
// entry, say - the allowlist would refuse the whole archive and every cache would miss.
func TestWriteTarballFrom_RoundTripsThroughTheAllowlist(t *testing.T) {
	source := t.TempDir()
	first := filepath.Join(source, "node_modules")
	second := filepath.Join(source, "m2")
	require.NoError(t, os.MkdirAll(filepath.Join(first, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(second, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(first, "pkg", "index.js"), []byte("module"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(second, "dep.jar"), []byte("jar"), 0o644))

	firstPath := filepath.ToSlash(first)
	secondPath := filepath.ToSlash(second)

	// Packed exactly as the cache save does: rooted at "/", with the cached paths as
	// both the patterns and the readable mounts.
	buf := &bytes.Buffer{}
	require.NoError(t, WriteTarballFrom(buf, "/", []string{firstPath, secondPath}, []string{firstPath, secondPath}))

	// Restored exactly as the cache restore does, with those same paths as the allowlist.
	require.NoError(t, UnpackTarball("/", buf, WithAllowedRoots(firstPath, secondPath)))

	restored, err := os.ReadFile(filepath.Join(first, "pkg", "index.js"))
	require.NoError(t, err)
	assert.Equal(t, "module", string(restored))

	restored, err = os.ReadFile(filepath.Join(second, "dep.jar"))
	require.NoError(t, err)
	assert.Equal(t, "jar", string(restored))
}
