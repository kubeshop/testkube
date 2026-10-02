package executioncache

import (
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestRestoreCandidateMatchesTheListingPolicy pins that streaming selects the same entry
// MatchRestore would have, so the bound can be removed without changing what a restore
// resolves to.
func TestRestoreCandidateMatchesTheListingPolicy(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Key: "npm-old", LastModified: now.Add(-2 * time.Hour)},
		{Key: "npm-new", LastModified: now},
		{Key: "npm-mid", LastModified: now.Add(-time.Hour)},
		{Key: "yarn-x", LastModified: now.Add(time.Hour)},
		{Key: "nope", LastModified: now.Add(48 * time.Hour)},
	}

	t.Run("newest within the prefix wins, whatever order it arrives in", func(t *testing.T) {
		candidate := NewRestoreCandidate("npm-")
		for _, entry := range entries {
			candidate.Consider(entry)
		}
		best, found := candidate.Best()
		require.True(t, found)
		assert.Equal(t, "npm-new", best.Key)

		// The same set offered in reverse resolves identically, which is what makes the
		// order a store happens to list in irrelevant.
		reversed := NewRestoreCandidate("npm-")
		for i := len(entries) - 1; i >= 0; i-- {
			reversed.Consider(entries[i])
		}
		other, _ := reversed.Best()
		assert.Equal(t, best.Key, other.Key)
	})

	t.Run("it agrees with MatchRestore", func(t *testing.T) {
		// The property that lets one replace the other for the prefix half of the policy.
		for _, prefix := range []string{"npm-", "yarn-", "absent-"} {
			candidate := NewRestoreCandidate(prefix)
			for _, entry := range entries {
				candidate.Consider(entry)
			}
			streamed, streamedFound := candidate.Best()

			listed, _, listedFound := MatchRestore(entries, "no-exact-match", []string{prefix})
			assert.Equal(t, listedFound, streamedFound, "prefix %q", prefix)
			assert.Equal(t, listed.Key, streamed.Key, "prefix %q", prefix)
		}
	})

	t.Run("an empty prefix matches nothing", func(t *testing.T) {
		candidate := NewRestoreCandidate("")
		for _, entry := range entries {
			candidate.Consider(entry)
		}
		_, found := candidate.Best()
		assert.False(t, found, "an empty prefix would be the whole scope")
	})

	t.Run("nothing offered is not a match", func(t *testing.T) {
		_, found := NewRestoreCandidate("npm-").Best()
		assert.False(t, found)
	})
}

// TestRestoreCandidateRefusesToGuessPastTheCap covers what happens when a prefix is too
// broad to choose from.
//
// The cap exists so one restore cannot read an unbounded number of objects. It must not
// decide the answer, though: entries arrive in the store's lexical order while the choice
// is made on recency, so the best of a truncated run is not the best of the prefix.
// Returning it would be a restore of the wrong dependency set, reported as a hit and
// repeated until the entry expired - where a miss only costs a reinstall.
func TestRestoreCandidateRefusesToGuessPastTheCap(t *testing.T) {
	newest := Entry{Key: "npm-zzz", LastModified: time.Now()}

	t.Run("under the cap it answers normally", func(t *testing.T) {
		candidate := NewRestoreCandidate("npm-")
		for i := 0; i < 10; i++ {
			require.True(t, candidate.Consider(Entry{
				Key:          "npm-" + strconv.Itoa(i),
				LastModified: time.Now().Add(-time.Hour),
			}), "the cap is nowhere near")
		}
		require.True(t, candidate.Consider(newest))

		best, found := candidate.Best()
		require.True(t, found)
		assert.Equal(t, newest.Key, best.Key)
		assert.False(t, candidate.Overflowed())
	})

	t.Run("past the cap it stops reading and reports no answer", func(t *testing.T) {
		candidate := NewRestoreCandidate("npm-")
		older := Entry{Key: "npm-aaa", LastModified: time.Now().Add(-time.Hour)}

		stopped := 0
		for i := 0; i < MaxRestoreCandidates+10; i++ {
			if !candidate.Consider(older) {
				stopped = i
				break
			}
		}
		require.Equal(t, MaxRestoreCandidates, stopped,
			"it should ask the caller to stop exactly at the cap, not read past it")

		_, found := candidate.Best()
		assert.False(t, found, "a truncated read cannot name the newest entry, so it names none")
		assert.True(t, candidate.Overflowed(), "and says why, so the cause is not mistaken for an empty prefix")
	})

	t.Run("entries outside the prefix still count", func(t *testing.T) {
		// The cap bounds what the store is asked to produce, and an entry costs the same
		// to read whether or not it matches.
		candidate := NewRestoreCandidate("npm-")
		for i := 0; i < MaxRestoreCandidates; i++ {
			candidate.Consider(Entry{Key: "yarn-" + strconv.Itoa(i)})
		}
		assert.False(t, candidate.Consider(newest), "the cap is reached on volume read, not on matches")
	})
}

// TestMaxRestoreKeysMatchesTheCRD is the only thing holding the two declarations of this
// limit together.
//
// The CRD marker is what a cluster enforces on a workflow; MaxRestoreKeys is what a
// control plane enforces on the request the agent sends. If they drift apart the looser
// one decides, and the tighter one becomes a rejection of workflows the schema accepted.
// This package cannot import the API types - they are a layer above it - so the marker is
// read from the source that carries it.
func TestMaxRestoreKeysMatchesTheCRD(t *testing.T) {
	const typesPath = "../../api/testworkflows/v1/step_types.go"

	source, err := os.ReadFile(typesPath)
	require.NoError(t, err, "the API types are where the CRD marker lives")

	field := regexp.MustCompile(`\+kubebuilder:validation:MaxItems=(\d+)\s*\n\s*RestoreKeys\b`)
	found := field.FindSubmatch(source)
	require.NotNil(t, found, "RestoreKeys must carry a MaxItems marker directly above it in %s", typesPath)

	declared, err := strconv.Atoi(string(found[1]))
	require.NoError(t, err)
	assert.Equal(t, MaxRestoreKeys, declared,
		"the CRD admits %d restore keys and the control planes admit %d", declared, MaxRestoreKeys)
}
