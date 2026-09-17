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
