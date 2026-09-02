// Package executioncache derives where a step's dependency cache lives in object
// storage, and picks which stored entry a restore should use.
//
// Both halves live here, away from any transport, for one reason: the control plane in
// this repository and the commercial one must agree on them exactly. A disagreement
// would not fail loudly - it would produce entries that the other plane can never find,
// so every run would miss its own cache and quietly reinstall everything. The
// commercial plane is expected to import this package rather than reimplement it.
package executioncache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// ObjectPrefix is the top-level folder every cache entry lives under.
	//
	// It gives the bucket lifecycle rule something to filter on, and keeps entries out
	// of the artifact namespace, which is keyed by execution id. An execution id can
	// never collide with it: ids do not begin with a dot.
	ObjectPrefix = ".tkcache/v1"

	// ObjectSuffix marks a stored entry. Kept out of the encoded key so that a prefix
	// query for a restore key cannot accidentally match it.
	ObjectSuffix = ".tar.gz"

	// MaxKeyBytes caps a cache key before encoding.
	//
	// S3 object keys are limited to 1024 bytes and encoding can triple a byte, so the
	// cap has to leave room for the prefix, the scope segments and the suffix. It is
	// also a correctness limit rather than only a storage one: two keys that differ
	// beyond a silent truncation point would share an entry.
	MaxKeyBytes = 512

	// maxSegmentChars bounds a sanitized name segment before its disambiguating hash.
	maxSegmentChars = 48
)

// Scope is how widely a cache entry is shared.
type Scope string

const (
	// ScopeWorkflow shares an entry only with other executions of the same workflow.
	ScopeWorkflow Scope = "workflow"
	// ScopeEnvironment shares an entry with every workflow in the environment.
	ScopeEnvironment Scope = "environment"
)

// ParseScope resolves the scope a step declared.
//
// An empty or unrecognised value resolves to the narrowest scope, never the widest: a
// field this layer fails to understand must not silently widen who can write what a
// workflow will later execute.
func ParseScope(value string) Scope {
	if Scope(value) == ScopeEnvironment {
		return ScopeEnvironment
	}
	return ScopeWorkflow
}

// ValidateKey reports whether a key may be used at all.
//
// The key reaches us as the value of an expression evaluated inside the pod, so it is
// arbitrary text and worth checking before it becomes part of an object name. An empty
// key is rejected on its own account: it is what an unmatched hash_files() produces, and
// caching every such step under one shared entry is exactly the wrong answer.
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("cache key is empty")
	}
	if len(key) > MaxKeyBytes {
		return fmt.Errorf("cache key is %d bytes, over the %d byte limit", len(key), MaxKeyBytes)
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("cache key contains a control character")
		}
	}
	return nil
}

// encodeKey turns a key into one path segment that cannot express a path.
//
// Percent-encoding rather than hashing, because it is prefix-preserving: every character
// encodes independently, so a prefix of the key encodes to a prefix of the segment. That
// is what lets a restore key be answered by a plain prefix query instead of a manifest
// object that would need read-modify-write. It is chosen over hex for the same
// prefix property plus one practical gain - the keys stay readable in a bucket listing,
// which matters the first time somebody debugs a cache that will not hit.
//
// '/' becomes %2F and '.' is escaped, so neither a separator nor a ".." segment can
// appear however the key was written.
func encodeKey(key string) string {
	var out strings.Builder
	out.Grow(len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			out.WriteByte(c)
		default:
			fmt.Fprintf(&out, "%%%02X", c)
		}
	}
	return out.String()
}

// sanitizeSegment turns a name into one path segment.
//
// The disambiguating hash is appended always, not only when the name was truncated:
// two names that sanitize to the same slug must not end up sharing a cache scope.
func sanitizeSegment(name string) string {
	var slug strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			slug.WriteByte(c)
		default:
			slug.WriteByte('_')
		}
	}
	trimmed := slug.String()
	if len(trimmed) > maxSegmentChars {
		trimmed = trimmed[:maxSegmentChars]
	}
	sum := sha256.Sum256([]byte(name))
	return trimmed + "-" + hex.EncodeToString(sum[:4])
}

// ScopePrefix is the folder holding every entry a given scope can see.
//
// Both arguments must come from the execution the caller has already authenticated,
// never from the request: deriving the workflow segment server-side is the whole reason
// a workflow-scoped entry cannot be read or written by another workflow.
func ScopePrefix(environmentID, workflowName string, scope Scope) string {
	env := sanitizeSegment(environmentID)
	if scope == ScopeEnvironment {
		return fmt.Sprintf("%s/%s/e", ObjectPrefix, env)
	}
	return fmt.Sprintf("%s/%s/w/%s", ObjectPrefix, env, sanitizeSegment(workflowName))
}

// ObjectName is the object holding the entry for an exact key.
func ObjectName(environmentID, workflowName string, scope Scope, key string) string {
	return ScopePrefix(environmentID, workflowName, scope) + "/" + encodeKey(key) + ObjectSuffix
}

// ObjectNamePrefix is the prefix matching every entry whose key starts with keyPrefix.
//
// Deliberately has no ObjectSuffix: a restore key matches on the start of a key, so the
// query has to stay open-ended.
func ObjectNamePrefix(environmentID, workflowName string, scope Scope, keyPrefix string) string {
	return ScopePrefix(environmentID, workflowName, scope) + "/" + encodeKey(keyPrefix)
}
