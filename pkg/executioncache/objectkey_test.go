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
var safeObject = regexp.MustCompile(`^\.tkcache/v1/[A-Za-z0-9_-]+/(e|w/[A-Za-z0-9_-]+)/[A-Za-z0-9_%-]+\.tar\.gz$`)

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
	assert.Regexp(t, `^[A-Za-z0-9_-]+$`, sanitizeSegment("../../etc/passwd"))
}

// TestParseScopeDefaultsToNarrowest: an unknown value must never widen sharing.
func TestParseScopeDefaultsToNarrowest(t *testing.T) {
	assert.Equal(t, ScopeEnvironment, ParseScope("environment"))
	assert.Equal(t, ScopeWorkflow, ParseScope("workflow"))
	assert.Equal(t, ScopeWorkflow, ParseScope(""))
	assert.Equal(t, ScopeWorkflow, ParseScope("Environment"))
	assert.Equal(t, ScopeWorkflow, ParseScope("anything-else"))
}
