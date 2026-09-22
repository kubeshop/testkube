package executioncache

import "time"

// Entry is a stored cache entry as a listing reports it.
type Entry struct {
	// Key is the object name, relative to nothing in particular - callers pass
	// whatever their listing returned and get one of these values back.
	Key string
	// Size is the compressed size in bytes.
	Size int64
	// LastModified is when the entry was written, which is how the newest of several
	// restore-key matches is chosen.
	LastModified time.Time
}

// MatchRestore picks the entry a restore should use, and is the whole matching policy.
//
// Order matters twice over, and the two orders are different:
//
//   - restoreKeys are tried in the order the workflow declared them, because that is
//     the author stating which fallback they prefer. The first one with any match wins,
//     even if a later one has a more recent entry.
//   - within a single restore key, the most recently written entry wins, because the
//     keys under one prefix are unordered and recency is the only sensible tiebreak.
//
// exact reports whether the entry is the one asked for, rather than a fallback. A caller
// uses it to decide whether saving afterwards would be redundant.
func MatchRestore(entries []Entry, exactName string, prefixes []string) (match Entry, exact bool, found bool) {
	for _, entry := range entries {
		if entry.Key == exactName {
			return entry, true, true
		}
	}

	for _, prefix := range prefixes {
		// An empty prefix would match the entire scope, which is never what a workflow
		// meant to ask for.
		if prefix == "" {
			continue
		}
		var best Entry
		var bestFound bool
		for _, entry := range entries {
			if len(entry.Key) < len(prefix) || entry.Key[:len(prefix)] != prefix {
				continue
			}
			if !bestFound || entry.LastModified.After(best.LastModified) {
				best, bestFound = entry, true
			}
		}
		if bestFound {
			return best, false, true
		}
	}

	return Entry{}, false, false
}

// RestoreCandidate picks the entry a single restore key resolves to, from entries offered
// one at a time rather than collected first.
//
// It exists because a listing has to be bounded somewhere, and bounding it by page was
// the wrong place: a store returns its page in lexical order while this policy selects on
// recency, so the newest entry under a busy prefix could sort past the end and a restore
// would silently take an older dependency set. Entries are immutable and leave only on
// expiry, so a prefix reaches that size in the ordinary course of things.
//
// Offering them one at a time means the caller holds one entry whatever the prefix
// contains, and the cap below bounds the reading rather than the choosing - so neither
// the answer nor the work is decided by where a page happened to end.
//
// The policy is the same one MatchRestore applies within a prefix, and is shared for the
// same reason the rest of this package is: the two control planes must agree on which
// entry a restore key selects, and a disagreement would not fail loudly.
type RestoreCandidate struct {
	prefix     string
	seen       int
	best       Entry
	found      bool
	overflowed bool
}

// MaxRestoreCandidates bounds how many entries one restore key may be chosen from.
//
// A restore must not be an unbounded scan of the store - a prefix has no cardinality
// limit, and one lookup should not be able to read everything under it. But the cap
// cannot decide the answer either, so exceeding it produces a refusal rather than the
// best of what was read; see Best.
//
// Set far above any prefix a workflow produces on purpose. A key is content-derived, so a
// prefix accumulates roughly one entry per distinct dependency set within the expiry
// window; reaching six figures means the restore key matches far more than its author can
// have intended.
const MaxRestoreCandidates = 100_000

// MaxRestoreKeys bounds how many restore keys one lookup may offer.
//
// It is the limit the CRD already states on StepCache.RestoreKeys, restated here because
// the CRD bounds what a workflow may declare and this bounds what arrives: a restore is
// requested over the wire by the agent, which can put any number of keys in the request
// whatever the workflow said. Each key is a separate listing of the store, so without
// this one request turns into as many round trips as it cares to ask for.
//
// A control plane refuses a longer list rather than reading the first few, because a
// request that exceeds it did not come from a workflow this schema accepts, and silently
// answering part of it would hide that.
//
// Keep in step with the kubebuilder MaxItems marker on StepCache.RestoreKeys. The two
// cannot be derived from one another - the API types are a layer this package must not
// depend on - so a change to either is a change to both.
const MaxRestoreKeys = 10

// NewRestoreCandidate starts a selection for one restore key's object-name prefix.
func NewRestoreCandidate(prefix string) *RestoreCandidate {
	return &RestoreCandidate{prefix: prefix}
}

// Consider offers an entry, and reports whether more are wanted.
//
// Entries outside the prefix are ignored, so a caller may hand it everything a listing
// produced without filtering first - but they still count towards the cap, because the
// cap bounds what a single restore asks of the store and an entry costs the same to read
// whether or not it matches.
//
// A false return means stop: the caller should end the listing there.
func (c *RestoreCandidate) Consider(entry Entry) bool {
	if c.seen >= MaxRestoreCandidates {
		c.overflowed = true
		return false
	}
	c.seen++

	// An empty prefix would match the whole scope, which no workflow meant to ask for.
	if c.prefix == "" {
		return true
	}
	if len(entry.Key) < len(c.prefix) || entry.Key[:len(c.prefix)] != c.prefix {
		return true
	}
	if !c.found || entry.LastModified.After(c.best.LastModified) {
		c.best, c.found = entry, true
	}
	return true
}

// Best returns the most recently written entry offered under the prefix.
//
// A prefix that overflowed has no answer rather than a provisional one. Entries arrive in
// the store's order, which is lexical, while the choice is made on recency - so the best
// of a truncated run is not the best of the prefix, and returning it would be a restore
// of the wrong dependency set presented as a hit. A miss costs a reinstall; a wrong hit
// costs a build against dependencies nobody asked for, and lasts until the entry expires.
func (c *RestoreCandidate) Best() (Entry, bool) {
	if c.overflowed {
		return Entry{}, false
	}
	return c.best, c.found
}

// Overflowed reports that the prefix held more than the cap, so the miss Best reports is
// a refusal to guess rather than an absence. Worth surfacing to whoever wrote the
// workflow: it means a restore key matches more than it can usefully choose from.
func (c *RestoreCandidate) Overflowed() bool {
	return c.overflowed
}
