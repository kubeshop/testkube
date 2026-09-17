package executioncache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMatchRestore(t *testing.T) {
	now := time.Now()
	entry := func(key string, ageMinutes int) Entry {
		return Entry{Key: key, Size: 1, LastModified: now.Add(-time.Duration(ageMinutes) * time.Minute)}
	}

	t.Run("exact match wins over any fallback", func(t *testing.T) {
		// The exact entry is the oldest here on purpose: an exact key is a statement
		// about contents, so recency must not override it.
		match, exact, found := MatchRestore([]Entry{
			entry("npm-newer", 1),
			entry("npm-wanted", 99),
		}, "npm-wanted", []string{"npm-"})

		assert.True(t, found)
		assert.True(t, exact)
		assert.Equal(t, "npm-wanted", match.Key)
	})

	t.Run("newest wins within one restore key", func(t *testing.T) {
		match, exact, found := MatchRestore([]Entry{
			entry("npm-old", 90),
			entry("npm-newest", 2),
			entry("npm-middle", 30),
		}, "npm-absent", []string{"npm-"})

		assert.True(t, found)
		assert.False(t, exact)
		assert.Equal(t, "npm-newest", match.Key)
	})

	t.Run("declared restore key order beats recency across keys", func(t *testing.T) {
		// "deps-" holds a much newer entry, but the workflow asked for "npm-" first,
		// and that preference is the author's to make.
		match, _, found := MatchRestore([]Entry{
			entry("deps-brand-new", 1),
			entry("npm-stale", 500),
		}, "npm-absent", []string{"npm-", "deps-"})

		assert.True(t, found)
		assert.Equal(t, "npm-stale", match.Key)
	})

	t.Run("falls through to a later restore key", func(t *testing.T) {
		match, _, found := MatchRestore([]Entry{
			entry("deps-found", 5),
		}, "npm-absent", []string{"npm-", "deps-"})

		assert.True(t, found)
		assert.Equal(t, "deps-found", match.Key)
	})

	t.Run("no match is a miss, not an error", func(t *testing.T) {
		_, exact, found := MatchRestore([]Entry{entry("pip-x", 1)}, "npm-absent", []string{"npm-"})
		assert.False(t, found)
		assert.False(t, exact)
	})

	t.Run("empty listing", func(t *testing.T) {
		_, _, found := MatchRestore(nil, "npm-absent", []string{"npm-"})
		assert.False(t, found)
	})

	t.Run("no restore keys means exact or nothing", func(t *testing.T) {
		_, _, found := MatchRestore([]Entry{entry("npm-other", 1)}, "npm-absent", nil)
		assert.False(t, found)
	})

	// An empty restore key would match every entry in the scope, which no workflow
	// meant to request - and under scope: environment that is another team's cache.
	t.Run("empty restore key is skipped", func(t *testing.T) {
		_, _, found := MatchRestore([]Entry{entry("anything", 1)}, "npm-absent", []string{""})
		assert.False(t, found)
	})

	t.Run("empty restore key does not shadow a real one", func(t *testing.T) {
		match, _, found := MatchRestore([]Entry{
			entry("anything", 1),
			entry("npm-real", 5),
		}, "npm-absent", []string{"", "npm-"})

		assert.True(t, found)
		assert.Equal(t, "npm-real", match.Key)
	})

	t.Run("prefix must match from the start", func(t *testing.T) {
		_, _, found := MatchRestore([]Entry{entry("x-npm-y", 1)}, "npm-absent", []string{"npm-"})
		assert.False(t, found)
	})
}
