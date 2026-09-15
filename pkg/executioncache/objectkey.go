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
	"strconv"
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

	// MaxEncodedKeyBytes caps what a key becomes rather than what it started as.
	//
	// The limit belongs to the store: an S3 object key may be 1024 bytes. A cache key is
	// percent-encoded into one segment of that name, and encoding can triple a byte, so
	// a cap on the key before encoding does not bound the result - 512 raw bytes of
	// punctuation encode to 1536, which no prefix can make fit. Measuring the encoded
	// form is exact where measuring the input was a guess.
	//
	// The budget is what remains of 1024 after the longest layout this plane produces:
	// ".tkcache/v1/" and "/testworkflows/" and a separator and ".tar.gz" come to 35, and
	// two name segments can be 253 each, leaving 483. Rounded down to leave the
	// commercial plane, which nests deeper, a little room to share this constant.
	MaxEncodedKeyBytes = 480

	// MaxKeyBytes caps the key before encoding as well, so an absurd input is refused
	// before any work is done on it. It is the same number because a key of unreserved
	// characters encodes one to one: that is the case where the raw length is the
	// encoded length, and no raw key shorter than this can be under the encoded cap by
	// virtue of its length alone.
	MaxKeyBytes = MaxEncodedKeyBytes

	// maxSegmentChars bounds a name used verbatim as a path segment. A Kubernetes name
	// cannot exceed 253 characters, so this only ever rejects something that was not one.
	maxSegmentChars = 253

	// workflowScopeSegment and sharedScopeSegment name the two scopes in an object name.
	//
	// Spelled out rather than abbreviated, and spelled the same way the commercial
	// control plane spells them, so that an operator looking into either bucket meets
	// one vocabulary instead of two. The layouts around them stay different on purpose -
	// that plane is multi-tenant and keys by organization - but there is no reason for
	// the words to differ as well.
	workflowScopeSegment = "testworkflows"
	sharedScopeSegment   = "shared"
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
	// Checked on the encoded form, because that is what has to fit inside an object
	// name. The raw cap above does not bound it: a key of punctuation triples.
	if encoded := len(EncodeKey(key)); encoded > MaxEncodedKeyBytes {
		return fmt.Errorf("cache key encodes to %d bytes, over the %d byte limit", encoded, MaxEncodedKeyBytes)
	}
	return nil
}

// EncodeKey turns a key into one path segment that cannot express a path.
//
// Exported because both control planes need it and neither may reimplement it. They do
// not share a storage layout - the commercial one is multi-tenant, keyed by organization
// and environment - but they must encode a key the same way, because this is the part
// that keeps attacker-influenced text from becoming a path, and two encoders would be
// two chances to get that wrong.
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
func EncodeKey(key string) string {
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

// sanitizeSegment turns a name into one path segment, using the name itself wherever it
// already is one - which is always, for the names this actually receives.
//
// A workflow name is a Kubernetes object name: lowercase alphanumeric with '-' and '.',
// at most 253 characters. It contains no separator, cannot be "." or "..", and is unique
// within its environment. Transforming it bought nothing and cost the one thing these
// names are for, which is an operator reading a bucket listing to work out why a cache
// does not hit.
//
// This used to slug the name, cut it at 48 characters and append a digest. The digest
// existed because the cut created collisions - two names agreeing on their first 48
// characters became one segment, and two workflows sharing a scope is one restoring a
// dependency tree the other wrote. Removing the cut removes the reason for the digest
// rather than weakening anything: distinct names now produce distinct segments because
// they are distinct.
//
// The fallback stays for input that is not a name of that kind, which in practice means
// a misconfigured environment id rather than a workflow. It is a full digest rather than
// a partial one, and it is prefixed with an underscore - which no Kubernetes name may
// begin with - so a real name can neither be mistaken for a fallback nor collide with one.
func sanitizeSegment(name string) string {
	if isSafeSegment(name) {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	return "_" + hex.EncodeToString(sum[:])
}

// isSafeSegment reports whether a name can be used as a path segment as it stands.
func isSafeSegment(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > maxSegmentChars {
		return false
	}
	if strings.HasPrefix(name, "_") {
		// Reserved for the fallback, so the two can never be confused.
		return false
	}
	for _, r := range name {
		if r == '/' || r == '\\' || r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// ScopePrefix is the folder holding every entry a given scope can see.
//
// Both arguments must come from the execution the caller has already authenticated,
// never from the request: deriving the workflow segment server-side is the whole reason
// a workflow-scoped entry cannot be read or written by another workflow.
func ScopePrefix(environmentID, workflowName string, scope Scope) string {
	env := sanitizeSegment(environmentID)
	if scope == ScopeEnvironment {
		return fmt.Sprintf("%s/%s/%s", ObjectPrefix, env, sharedScopeSegment)
	}
	return fmt.Sprintf("%s/%s/%s/%s", ObjectPrefix, env, workflowScopeSegment, sanitizeSegment(workflowName))
}

// ObjectName is the object holding the entry for an exact key.
func ObjectName(environmentID, workflowName string, scope Scope, key string) string {
	return ScopePrefix(environmentID, workflowName, scope) + "/" + EncodeKey(key) + ObjectSuffix
}

// ObjectNamePrefix is the prefix matching every entry whose key starts with keyPrefix.
//
// Deliberately has no ObjectSuffix: a restore key matches on the start of a key, so the
// query has to stay open-ended.
func ObjectNamePrefix(environmentID, workflowName string, scope Scope, keyPrefix string) string {
	return ScopePrefix(environmentID, workflowName, scope) + "/" + EncodeKey(keyPrefix)
}

// KeyFromObjectName recovers the key a workflow wrote from a stored object's name, so
// that a restore can report which entry it used in the terms the author recognises
// rather than as an encoded path.
//
// An object this package did not write is returned as-is: reporting the raw name is more
// useful than reporting nothing, and it is not worth an error on a reporting path.
func KeyFromObjectName(prefix, objectName string) string {
	encoded := strings.TrimPrefix(objectName, prefix+"/")
	encoded = strings.TrimSuffix(encoded, ObjectSuffix)

	var out strings.Builder
	out.Grow(len(encoded))
	for i := 0; i < len(encoded); i++ {
		if encoded[i] != '%' {
			out.WriteByte(encoded[i])
			continue
		}
		if i+2 >= len(encoded) {
			return encoded
		}
		value, err := strconv.ParseUint(encoded[i+1:i+3], 16, 8)
		if err != nil {
			return encoded
		}
		out.WriteByte(byte(value))
		i += 2
	}
	return out.String()
}
