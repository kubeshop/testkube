package executioncache

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeObject is the shape every derived object name must have: the fixed prefix, one
// environment segment, the scope discriminator, an optional workflow segment, then a
// single encoded key segment. Nothing else may introduce a '/' or a '.'.
// A name segment may now carry a dot, because a Kubernetes name may and is used as it
// stands. The key segment is still percent-encoded, so it can only ever be the
// unreserved set plus the escapes.
var safeObject = regexp.MustCompile(`^\.tkcache/v1/(_[0-9a-f]{64}|[A-Za-z0-9._-]+)/(shared|testworkflows/(_[0-9a-f]{64}|[A-Za-z0-9._-]+))/[A-Za-z0-9_%-]+\.tar\.gz$`)

// TestObjectNameConfinesHostileKeys is the load-bearing test of the whole layout.
//
// The key is the value of an expression evaluated in the pod, so it is arbitrary text.
// If any of these could put a '/' or a '..' into the object name, a workflow-scoped
// entry could be written outside its own scope and the prefix-filtered lifecycle rule
// would stop covering it.
func TestObjectNameConfinesHostileKeys(t *testing.T) {
	hostile := []string{
		"../../e/shared",
		"a/b",
		"/absolute",
		"..",
		".",
		"....//....//x",
		"key with spaces",
		"key\nnewline",
		"npm-你好",
		"%2e%2e%2f",
		"%",
		strings.Repeat("k", MaxKeyBytes),
	}

	for _, key := range hostile {
		t.Run(key, func(t *testing.T) {
			for _, scope := range []Scope{ScopeWorkflow, ScopeEnvironment} {
				name := ObjectName("env-1", "wf-1", scope, key)

				assert.Regexp(t, safeObject, name)
				assert.NotContains(t, strings.TrimSuffix(name, ObjectSuffix), "..")
				assert.LessOrEqual(t, len(name), 1024, "object name must fit an S3 key")

				// Beyond the fixed structure, the key contributes no separators at all.
				body := strings.TrimPrefix(name, ScopePrefix("env-1", "wf-1", scope)+"/")
				assert.NotContains(t, body, "/")
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	assert.NoError(t, ValidateKey("npm-abc"))
	assert.NoError(t, ValidateKey(strings.Repeat("k", MaxKeyBytes)))

	// An unmatched hash_files() yields "". Caching every such step under one shared
	// entry is worse than not caching, so it is refused rather than normalised.
	assert.ErrorContains(t, ValidateKey(""), "empty")
	assert.ErrorContains(t, ValidateKey(strings.Repeat("k", MaxKeyBytes+1)), "over the")
	assert.ErrorContains(t, ValidateKey("npm-\x00abc"), "control character")
	assert.ErrorContains(t, ValidateKey("npm-\tabc"), "control character")
}

// TestEncodeKeyPreservesPrefixes is what makes restoreKeys answerable by a prefix
// query. If the encoding ever stopped being prefix-preserving, restore keys would
// silently stop matching and every fallback would turn into a miss.
func TestEncodeKeyPreservesPrefixes(t *testing.T) {
	pairs := [][2]string{
		{"npm-", "npm-abc123"},
		{"npm-", "npm-"},
		{"deps/", "deps/v2/abc"},
		{"a b", "a b c"},
		{"%", "%2e"},
		{"你", "你好"},
	}

	for _, pair := range pairs {
		prefix, full := pair[0], pair[1]
		require.True(t, strings.HasPrefix(full, prefix), "test data must itself be a prefix")

		queried := ObjectNamePrefix("env-1", "wf-1", ScopeWorkflow, prefix)
		stored := ObjectName("env-1", "wf-1", ScopeWorkflow, full)
		assert.True(t, strings.HasPrefix(stored, queried),
			"%q should be found by a query for %q", full, prefix)
	}

	// And a key that merely looks similar must not be matched.
	stored := ObjectName("env-1", "wf-1", ScopeWorkflow, "pip-abc")
	queried := ObjectNamePrefix("env-1", "wf-1", ScopeWorkflow, "npm-")
	assert.False(t, strings.HasPrefix(stored, queried))
}

// TestScopeIsolation is the executable form of the promise that scope: workflow makes.
func TestScopeIsolation(t *testing.T) {
	a := ScopePrefix("env-1", "workflow-a", ScopeWorkflow)
	b := ScopePrefix("env-1", "workflow-b", ScopeWorkflow)
	shared := ScopePrefix("env-1", "workflow-a", ScopeEnvironment)

	assert.NotEqual(t, a, b)
	// Neither may contain the other, or a prefix query in one would reach into the other.
	assert.False(t, strings.HasPrefix(a, b))
	assert.False(t, strings.HasPrefix(b, a))
	assert.False(t, strings.HasPrefix(a, shared))
	assert.False(t, strings.HasPrefix(shared, a))

	// The environment scope ignores the workflow entirely, so two workflows share it.
	assert.Equal(t, shared, ScopePrefix("env-1", "workflow-b", ScopeEnvironment))

	// A different environment is a different scope even for the same workflow.
	assert.NotEqual(t, a, ScopePrefix("env-2", "workflow-a", ScopeWorkflow))
}

// TestSanitizeSegmentDisambiguates covers names that reduce to the same slug: without
// the appended hash they would silently share one cache scope.
func TestSanitizeSegmentDisambiguates(t *testing.T) {
	assert.NotEqual(t, sanitizeSegment("my workflow"), sanitizeSegment("my/workflow"))
	assert.NotEqual(t, sanitizeSegment("a.b"), sanitizeSegment("a_b"))
	assert.NotEqual(t,
		sanitizeSegment(strings.Repeat("x", 60)+"one"),
		sanitizeSegment(strings.Repeat("x", 60)+"two"))
	assert.Equal(t, sanitizeSegment("stable"), sanitizeSegment("stable"))
	assert.Regexp(t, `^[A-Za-z0-9._-]+$`, sanitizeSegment("../../etc/passwd"))
}

// TestParseScopeDefaultsToNarrowest: an unknown value must never widen sharing.
func TestParseScopeDefaultsToNarrowest(t *testing.T) {
	assert.Equal(t, ScopeEnvironment, ParseScope("environment"))
	assert.Equal(t, ScopeWorkflow, ParseScope("workflow"))
	assert.Equal(t, ScopeWorkflow, ParseScope(""))
	assert.Equal(t, ScopeWorkflow, ParseScope("Environment"))
	assert.Equal(t, ScopeWorkflow, ParseScope("anything-else"))
}

// TestKeyFromObjectNameRoundTrips covers the reporting path: a restore says which entry
// it used, and it has only the object name to say it from.
func TestKeyFromObjectNameRoundTrips(t *testing.T) {
	keys := []string{
		"npm-abc123",
		"npm-{{weird}}",
		"a/b",
		"../../e/shared",
		"key with spaces",
		"npm-你好",
		"100%",
		"-_.~",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			for _, scope := range []Scope{ScopeWorkflow, ScopeEnvironment} {
				prefix := ScopePrefix("env-1", "wf-1", scope)
				name := ObjectName("env-1", "wf-1", scope, key)
				assert.Equal(t, key, KeyFromObjectName(prefix, name))
			}
		})
	}
}

// TestKeyFromObjectNameToleratesForeignNames: the reporting path must not panic or lie
// loudly on an object some other writer put in the bucket.
func TestKeyFromObjectNameToleratesForeignNames(t *testing.T) {
	prefix := ScopePrefix("env-1", "wf-1", ScopeWorkflow)

	// Truncated escape, invalid hex, and a name outside the prefix entirely.
	assert.Equal(t, "abc%", KeyFromObjectName(prefix, prefix+"/abc%"+ObjectSuffix))
	assert.Equal(t, "abc%zz", KeyFromObjectName(prefix, prefix+"/abc%zz"+ObjectSuffix))
	assert.Equal(t, "somewhere/else", KeyFromObjectName(prefix, "somewhere/else"))
}

// TestEncodeKeyIsSafeAndPrefixPreserving covers EncodeKey directly, because it is the
// seam both control planes share. They keep their own storage layouts - the commercial
// one is keyed by organization and environment - but a second implementation of this
// would be a second chance to let attacker-influenced text become a path.
func TestEncodeKeyIsSafeAndPrefixPreserving(t *testing.T) {
	safeSegment := regexp.MustCompile(`^[A-Za-z0-9_%-]*$`)

	for _, key := range []string{
		"npm-abc",
		"../../e/shared",
		"a/b",
		"/absolute",
		"..",
		".",
		"key with spaces",
		"npm-你好",
		"100%",
	} {
		encoded := EncodeKey(key)
		assert.Regexp(t, safeSegment, encoded, "%q must encode to one path segment", key)
		assert.NotContains(t, encoded, "/", "%q", key)
		assert.NotContains(t, encoded, ".", "%q", key)
	}

	// Prefix-preserving, which is what lets a restore key be a plain prefix query.
	assert.True(t, strings.HasPrefix(EncodeKey("npm-abc"), EncodeKey("npm-")))
	assert.False(t, strings.HasPrefix(EncodeKey("pip-abc"), EncodeKey("npm-")))

	// And round-trips, so a plane can report the key an author wrote.
	assert.Equal(t, "../../e/shared", KeyFromObjectName("", EncodeKey("../../e/shared")))
}

// TestSanitizeSegmentKeepsNamesReadable pins what a scope prefix looks like, because the
// reason to keep it readable is that somebody is reading it - a bucket listing is where
// you go to work out why a cache does not hit.
//
// A Kubernetes name is already one safe path segment, so it is used as it stands. Two
// distinct names therefore produce two distinct segments by virtue of being distinct,
// which is a stronger separation than the digest this replaced: that one slugged the
// name and cut it at 48 characters, so names agreeing on their first 48 collided and
// needed the digest to tell them apart again.
func TestSanitizeSegmentKeepsNamesReadable(t *testing.T) {
	// Long, and sharing far more than a digest-free scheme could have tolerated before.
	first := "nightly-integration-suite-for-payments-service-eu-west-1"
	second := "nightly-integration-suite-for-payments-service-eu-west-2"

	assert.Equal(t, first, sanitizeSegment(first), "a Kubernetes name is used as it stands")
	assert.NotEqual(t, sanitizeSegment(first), sanitizeSegment(second))
	assert.Contains(t, ScopePrefix("env-1", first, ScopeWorkflow), first,
		"the name has to be legible in the object path")
}

// TestSanitizeSegmentContainsUnsafeNames covers the input that is not a Kubernetes name,
// which in practice means a misconfigured environment id rather than a workflow.
//
// It has to stay one segment whatever it contains, or a scope could be escaped. The
// fallback is a full digest behind an underscore - a character no Kubernetes name may
// begin with - so a real name can neither be mistaken for a fallback nor collide with one.
func TestSanitizeSegmentContainsUnsafeNames(t *testing.T) {
	for _, unsafe := range []string{
		"../../etc/passwd",
		"a/b",
		"..",
		".",
		"",
		"has\\backslash",
		"_starts-with-underscore",
		strings.Repeat("x", maxSegmentChars+1),
	} {
		got := sanitizeSegment(unsafe)
		assert.Regexp(t, `^_[0-9a-f]{64}$`, got, "%q has to fall back to a digest", unsafe)
		assert.NotContains(t, got, "/", "%q must not stay able to express a path", unsafe)
	}

	// Distinct unsafe names stay distinct, and none of them can reach a scope that a
	// real name occupies.
	assert.NotEqual(t, sanitizeSegment("a/b"), sanitizeSegment("a\\b"))
	assert.NotEqual(t, sanitizeSegment(".."), sanitizeSegment("."))
}

// TestValidateKeyBoundsTheEncodedLength covers the limit that belongs to the store.
//
// A key is percent-encoded into one segment of the object name, and encoding can triple
// a byte, so a cap on the key before encoding does not bound what reaches S3 - the
// earlier 512-byte raw cap admitted keys that encode to 1536.
func TestValidateKeyBoundsTheEncodedLength(t *testing.T) {
	// Encodes one-to-one, so the raw length is the encoded length.
	assert.NoError(t, ValidateKey(strings.Repeat("a", MaxEncodedKeyBytes)))

	// Triples, so it is well under the raw cap and well over the encoded one - which is
	// exactly the case the old check let through.
	punctuation := strings.Repeat("/", 200)
	require.LessOrEqual(t, len(punctuation), MaxKeyBytes,
		"the fixture has to pass the raw cap, or it is not testing the encoded one")
	err := ValidateKey(punctuation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encodes to")
}
