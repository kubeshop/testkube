package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	// Readlink reports the target with the host's separators, so compare in one form.
	assert.Equal(t, "../pkg/real.js", filepath.ToSlash(target))
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
//
// The packed files are deleted before unpacking, which is the whole point. Reading them
// back without deleting them proves nothing: they were already there, so the assertions
// passed against an empty archive for as long as this test existed in that form.
func TestWriteTarballFrom_RoundTripsThroughTheAllowlist(t *testing.T) {
	source := t.TempDir()
	if filepath.VolumeName(source) != "" {
		t.Skip("test assumes POSIX-rooted absolute paths; Windows drive paths won't match the '/'-rooted walker")
	}
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
	entries, err := WriteTarballFrom(buf, "/", cachePatternsFor(firstPath, secondPath), []string{firstPath, secondPath})
	require.NoError(t, err)
	require.Equal(t, 2, entries, "both cached files have to be in the archive")

	require.NoError(t, os.RemoveAll(first))
	require.NoError(t, os.RemoveAll(second))

	// Restored exactly as the cache restore does, with those same paths as the allowlist.
	require.NoError(t, UnpackTarball("/", buf, WithAllowedRoots(firstPath, secondPath)))

	restored, err := os.ReadFile(filepath.Join(first, "pkg", "index.js"))
	require.NoError(t, err)
	assert.Equal(t, "module", string(restored))

	restored, err = os.ReadFile(filepath.Join(second, "dep.jar"))
	require.NoError(t, err)
	assert.Equal(t, "jar", string(restored))
}

// cachePatternsFor mirrors cachePackPatterns in the toolkit's cache command, which is
// what turns a cached directory into patterns the walker can match. Duplicated rather
// than imported because that package imports this one.
func cachePatternsFor(paths ...string) []string {
	patterns := make([]string, 0, len(paths)*2)
	for _, declared := range paths {
		patterns = append(patterns, declared, path.Join(declared, "**"))
	}
	return patterns
}

// TestWriteTarballFrom_ABarePathMatchesNothing pins the walker behaviour that made the
// cache silently store empty archives, so that the workaround above is not mistaken for
// belt-and-braces and quietly dropped.
//
// The walker matches candidates against the patterns with doublestar and skips
// directories. A pattern naming a directory therefore matches the directory and nothing
// else - not one file inside it - so packing a cached path as written produces an empty
// archive with no error at all.
func TestWriteTarballFrom_ABarePathMatchesNothing(t *testing.T) {
	source := t.TempDir()
	if filepath.VolumeName(source) != "" {
		t.Skip("test assumes POSIX-rooted absolute paths; Windows drive paths won't match the '/'-rooted walker")
	}
	cached := filepath.Join(source, "cache-probe")
	require.NoError(t, os.MkdirAll(filepath.Join(cached, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cached, "marker"), []byte("m"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(cached, "sub", "deep"), []byte("d"), 0o644))

	cachedPath := filepath.ToSlash(cached)

	buf := &bytes.Buffer{}
	entries, err := WriteTarballFrom(buf, "/", []string{cachedPath}, []string{cachedPath})
	require.NoError(t, err, "the bare path fails silently rather than loudly, which is what made this hard to see")
	assert.Zero(t, entries, "a bare directory path matches only the directory, which is skipped")

	// With the pattern the cache actually uses, everything under the path is packed, at
	// any depth.
	buf = &bytes.Buffer{}
	entries, err = WriteTarballFrom(buf, "/", cachePatternsFor(cachedPath), []string{cachedPath})
	require.NoError(t, err)
	assert.Equal(t, 2, entries)

	names := tarballEntryNames(t, buf)
	assert.Contains(t, names, strings.TrimPrefix(cachedPath, "/")+"/marker")
	assert.Contains(t, names, strings.TrimPrefix(cachedPath, "/")+"/sub/deep")
}

// TestWriteTarballFrom_PacksASingleCachedFile is why the bare path is kept alongside
// "/**": that pattern does not match the path itself, so a cached path that names one
// file would otherwise pack nothing.
func TestWriteTarballFrom_PacksASingleCachedFile(t *testing.T) {
	source := t.TempDir()
	if filepath.VolumeName(source) != "" {
		t.Skip("test assumes POSIX-rooted absolute paths; Windows drive paths won't match the '/'-rooted walker")
	}
	cached := filepath.Join(source, "one.jar")
	require.NoError(t, os.WriteFile(cached, []byte("jar"), 0o644))

	cachedPath := filepath.ToSlash(cached)

	buf := &bytes.Buffer{}
	entries, err := WriteTarballFrom(buf, "/", cachePatternsFor(cachedPath), []string{filepath.ToSlash(source)})
	require.NoError(t, err)
	assert.Equal(t, 1, entries)
	assert.Contains(t, tarballEntryNames(t, buf), strings.TrimPrefix(cachedPath, "/"))
}

func tarballEntryNames(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()

	gz, err := gzip.NewReader(buf)
	require.NoError(t, err)
	defer gz.Close()

	var names []string
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err != nil {
			break
		}
		names = append(names, header.Name)
	}
	return names
}

// TestNormalizeEntryName pins the check that decides what an archive may name.
//
// The backslash cases are the ones that matter, and they are why this judges the name
// with `path` and never rewrites it with ToSlash. A tar entry name is slash-separated by
// the format, whatever host wrote the archive, so a backslash in it is a character in a
// filename rather than a separator - and it has to be read that way everywhere, or the
// same archive is judged differently depending on where it is unpacked.
func TestNormalizeEntryName(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		for raw, want := range map[string]string{
			"file.txt":          "file.txt",
			"sub/file.txt":      "sub/file.txt",
			"sub//file.txt":     "sub/file.txt",
			"./sub/file.txt":    "sub/file.txt",
			"sub/./file.txt":    "sub/file.txt",
			"deps/pkg/index.js": "deps/pkg/index.js",

			// One segment each, whose name merely contains backslashes. Refusing these
			// would refuse legitimate POSIX filenames; accepting them on one host and
			// not another is the inconsistency this check exists to remove.
			"\\abs.txt":      "\\abs.txt",
			"..\\escape.txt": "..\\escape.txt",
			"\\\\server\\x":  "\\\\server\\x",
		} {
			got, err := normalizeEntryName(raw)
			assert.NoError(t, err, "%q should be accepted", raw)
			assert.Equal(t, want, got, "%q", raw)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		for _, raw := range []string{
			"",
			"/abs.txt",
			"C:/abs.txt",
			"C:\\abs.txt", // drive-qualified, which is not a relative path anywhere
			"..",
			"../escape.txt",
			"sub/../../escape.txt",
			"sub/..",
			".",
		} {
			_, err := normalizeEntryName(raw)
			assert.ErrorContains(t, err, "unsafe file path", "%q should be rejected", raw)
		}
	})

	t.Run("the decision does not depend on the host", func(t *testing.T) {
		// The point of the change: the answer for a name is a property of the name. Run
		// through filepath, these two disagreed across platforms.
		_, err := normalizeEntryName("/abs.txt")
		assert.Error(t, err, "a slash-absolute name is absolute everywhere")

		got, err := normalizeEntryName("\\abs.txt")
		assert.NoError(t, err, "a backslash name is one filename everywhere")
		assert.Equal(t, "\\abs.txt", got, "and is passed on exactly as it was validated")
	})
}

// TestUnpackTarball_BackslashNamesPassTheNameCheck is the end-to-end form: a name holding
// backslashes is not rejected as a path, because it is not one.
//
// It asserts only that the name check let it through, not what landed on disk. What a
// backslash means to the filesystem is genuinely host-dependent - one filename on a
// POSIX host, a separator on Windows, where os.Root then refuses it - and that is the
// one part of this the validation deliberately does not try to paper over.
func TestUnpackTarball_BackslashNamesPassTheNameCheck(t *testing.T) {
	err := UnpackTarball(t.TempDir(), tarballOf(t,
		tar.Header{Name: "weird\\name.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
	))
	if err != nil {
		assert.NotContains(t, err.Error(), "unsafe file path",
			"a backslash in a tar name is a character, not a path to refuse")
	}
}

// TestUnpackTarball_RejectsDriveQualifiedNames covers the one backslash-adjacent form
// that is refused: a drive-qualified name is not relative on any host, and path.Join
// would quietly treat it as one.
func TestUnpackTarball_RejectsDriveQualifiedNames(t *testing.T) {
	for _, name := range []string{"C:\\abs.txt", "C:/abs.txt"} {
		t.Run(name, func(t *testing.T) {
			err := UnpackTarball(t.TempDir(), tarballOf(t,
				tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
			))
			assert.ErrorContains(t, err, "unsafe file path")
		})
	}
}
