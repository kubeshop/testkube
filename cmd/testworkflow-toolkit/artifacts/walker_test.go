package artifacts

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWalkerADirectoryPatternMatchesNothing pins the semantics the dependency cache was
// silently losing its contents to.
//
// The walker matches every candidate against the patterns and skips directories. So a
// pattern that names a directory matches the directory - which is skipped - and no file
// inside it, and the walk finishes with no error and nothing collected. Packing a cached
// path as the user wrote it therefore produced an empty archive: the save succeeded, the
// next run got a hit, and it restored nothing.
//
// Written against a synthetic filesystem so it decides the same thing on every host: the
// paths here are container paths, and the walker's own inputs go through filepath.
func TestWalkerADirectoryPatternMatchesNothing(t *testing.T) {
	fsys := fstest.MapFS{
		"data/cache-probe/marker":   {Data: []byte("m")},
		"data/cache-probe/sub/deep": {Data: []byte("d")},
		"data/elsewhere/other":      {Data: []byte("o")},
	}

	collect := func(patterns []string) []string {
		w := &walker{root: "/", searchPaths: []string{"/data/cache-probe"}, patterns: patterns}
		var seen []string
		require.NoError(t, w.Walk(fsys, func(path string, file fs.File, stat fs.FileInfo, err error) error {
			require.NoError(t, err)
			seen = append(seen, path)
			return nil
		}))
		return seen
	}

	assert.Empty(t, collect([]string{"/data/cache-probe"}),
		"a bare directory pattern matches the directory, which is skipped, and nothing else")

	// What the cache packs with instead. "/**" reaches every depth, and the bare path is
	// kept beside it for a cached path that names a single file - see cachePackPatterns.
	assert.ElementsMatch(t,
		[]string{"data/cache-probe/marker", "data/cache-probe/sub/deep"},
		collect([]string{"/data/cache-probe", "/data/cache-probe/**"}))
}

// TestWalkerAFilePatternMatchesTheFile is the other half: "/x/**" does not match "/x", so
// dropping the bare path would stop a single cached file from being packed at all.
func TestWalkerAFilePatternMatchesTheFile(t *testing.T) {
	fsys := fstest.MapFS{"data/one.jar": {Data: []byte("jar")}}

	w := &walker{root: "/", searchPaths: []string{"/data"}, patterns: []string{"/data/one.jar", "/data/one.jar/**"}}
	var seen []string
	require.NoError(t, w.Walk(fsys, func(path string, file fs.File, stat fs.FileInfo, err error) error {
		require.NoError(t, err)
		seen = append(seen, path)
		return nil
	}))

	assert.Equal(t, []string{"data/one.jar"}, seen)
}
