package executioncache

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// safeObject is the shape every derived object name must have: the fixed prefix, one
// environment segment, the scope discriminator, an optional workflow segment, the
// namespace, then a single encoded key segment. Nothing else may introduce a '/' or a '.'.
// A name segment may now carry a dot, because a Kubernetes name may and is used as it
// stands. The key segment is still percent-encoded, so it can only ever be the
// unreserved set plus the escapes.
//
// The namespace is pinned to the two forms that exist - the base one, and a pull
// request's - so a third one cannot appear here unnoticed.
var safeSegment = `(_[0-9a-f]{64}|[A-Za-z0-9._-]+)`
var safeObject = regexp.MustCompile(`^\.tkcache/v1/` + safeSegment +
	`/(shared|testworkflows/` + safeSegment + `)` +
	`/(base|pr-` + safeSegment + `)` +
	`/[A-Za-z0-9_%-]+\.tar\.gz$`)

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
				name := ObjectName("env-1", "wf-1", scope, BaseNamespace, key)

				assert.Regexp(t, safeObject, name)
				assert.NotContains(t, strings.TrimSuffix(name, ObjectSuffix), "..")
				assert.LessOrEqual(t, len(name), 1024, "object name must fit an S3 key")

				// Beyond the fixed structure, the key contributes no separators at all.
				body := strings.TrimPrefix(name, ScopePrefix("env-1", "wf-1", scope, BaseNamespace)+"/")
				assert.NotContains(t, body, "/")
			}
		})
	}
}

// TestPullRequestNamespaceConfinesHostileIdentifiers covers the other arbitrary text that
// now reaches an object name.
//
// The identifier is a pull request number or, failing that, its head ref - and a head ref
// is a branch name the pull request's author chose. A branch called `../base` must not be
// able to name the namespace a trusted run writes, which is the one thing the split exists
// to prevent.
func TestPullRequestNamespaceConfinesHostileIdentifiers(t *testing.T) {
	hostile := []string{
		"../base",
		"../../../base",
		"base",
		"a/b",
		"..",
		".",
		"",
		"pr-1",
		"_underscored",
		"ref with spaces",
		strings.Repeat("b", 300),
	}

	for _, identifier := range hostile {
		t.Run(identifier, func(t *testing.T) {
			namespace := PullRequestNamespace("", identifier)

			assert.NotEqual(t, BaseNamespace, namespace, "a pull request must never land in the base namespace")
			assert.NotContains(t, namespace, "/")
			assert.NotContains(t, namespace, "..")

			// And the whole object name still holds its shape.
			assert.Regexp(t, safeObject, ObjectName("env-1", "wf-1", ScopeWorkflow, namespace, "npm-abc"))
		})
	}

	// Distinct identifiers stay distinct, or two pull requests would share a namespace.
	assert.NotEqual(t, PullRequestNamespace("", "../base"), PullRequestNamespace("", "../../base"))
	assert.NotEqual(t, PullRequestNamespace("", "42"), PullRequestNamespace("", "43"))

	// A plain number reads as itself, which is the whole point of keeping names readable.
	assert.Equal(t, "pr-42", PullRequestNamespace("", "42"))
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

		queried := ObjectNamePrefix("env-1", "wf-1", ScopeWorkflow, BaseNamespace, prefix)
		stored := ObjectName("env-1", "wf-1", ScopeWorkflow, BaseNamespace, full)
		assert.True(t, strings.HasPrefix(stored, queried),
			"%q should be found by a query for %q", full, prefix)
	}

	// And a key that merely looks similar must not be matched.
	stored := ObjectName("env-1", "wf-1", ScopeWorkflow, BaseNamespace, "pip-abc")
	queried := ObjectNamePrefix("env-1", "wf-1", ScopeWorkflow, BaseNamespace, "npm-")
	assert.False(t, strings.HasPrefix(stored, queried))
}

// TestScopeIsolation is the executable form of the promise that scope: workflow makes.
func TestScopeIsolation(t *testing.T) {
	a := ScopePrefix("env-1", "workflow-a", ScopeWorkflow, BaseNamespace)
	b := ScopePrefix("env-1", "workflow-b", ScopeWorkflow, BaseNamespace)
	shared := ScopePrefix("env-1", "workflow-a", ScopeEnvironment, BaseNamespace)

	assert.NotEqual(t, a, b)
	// Neither may contain the other, or a prefix query in one would reach into the other.
	assert.False(t, strings.HasPrefix(a, b))
	assert.False(t, strings.HasPrefix(b, a))
	assert.False(t, strings.HasPrefix(a, shared))
	assert.False(t, strings.HasPrefix(shared, a))

	// The environment scope ignores the workflow entirely, so two workflows share it.
	assert.Equal(t, shared, ScopePrefix("env-1", "workflow-b", ScopeEnvironment, BaseNamespace))

	// A different environment is a different scope even for the same workflow.
	assert.NotEqual(t, a, ScopePrefix("env-2", "workflow-a", ScopeWorkflow, BaseNamespace))
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
				prefix := ScopePrefix("env-1", "wf-1", scope, BaseNamespace)
				name := ObjectName("env-1", "wf-1", scope, BaseNamespace, key)
				assert.Equal(t, key, KeyFromObjectName(prefix, name))
			}
		})
	}
}

// TestKeyFromObjectNameToleratesForeignNames: the reporting path must not panic or lie
// loudly on an object some other writer put in the bucket.
func TestKeyFromObjectNameToleratesForeignNames(t *testing.T) {
	prefix := ScopePrefix("env-1", "wf-1", ScopeWorkflow, BaseNamespace)

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
	assert.Contains(t, ScopePrefix("env-1", first, ScopeWorkflow, BaseNamespace), first,
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

// TestObjectNameFitsTheStore is the arithmetic behind MaxEncodedKeyBytes, asserted
// rather than described.
//
// It exists because the prose was once right and then quietly stopped being: the
// namespace segment was added to the layout without revisiting the budget, which left
// the worst case 254 bytes past the 1024 an S3 key may be. Nothing failed, because
// nothing checked - the keys validated, and only the upload would have been refused,
// long after the step had decided it had a cache.
//
// Every segment is at its own maximum here. That combination is unlikely, which is
// exactly why it needs a test rather than a reviewer.
func TestObjectNameFitsTheStore(t *testing.T) {
	const s3MaxKeyBytes = 1024

	name := strings.Repeat("a", maxSegmentChars)
	// Unreserved characters encode one to one, so this is both the longest key that
	// validates and the longest one that survives encoding.
	key := strings.Repeat("k", MaxEncodedKeyBytes)
	require.NoError(t, ValidateKey(key), "the longest accepted key has to be accepted")

	namespaces := map[string]string{
		"base":                   BaseNamespace,
		"pull request, short id": PullRequestNamespace("https://github.com/org/repo/pull/4294967295", "4294967295"),
		"pull request, hostile":  PullRequestNamespace("https://github.com/org/repo/pull/1", strings.Repeat("../", 1000)),
		"pull request, huge ref": PullRequestNamespace("https://github.com/org/repo/pull/2", strings.Repeat("b", 4096)),
		// The case that actually exercises maxNamespaceIdentifierChars. The two above
		// are hashed by the older segment rules whatever this cap says - one is not a
		// legal segment, the other is past maxSegmentChars - so neither would notice if
		// the cap went. A branch name of this length is an ordinary one.
		"pull request, long but legal ref": PullRequestNamespace("https://github.com/org/repo/pull/3", strings.Repeat("b", 200)),
	}

	for label, namespace := range namespaces {
		assert.LessOrEqual(t, len(namespace), maxNamespaceChars,
			"%s: the namespace has to stay inside the budget the key was sized against", label)

		for _, scope := range []Scope{ScopeWorkflow, ScopeEnvironment} {
			object := ObjectName(name, name, scope, namespace, key)
			assert.LessOrEqual(t, len(object), s3MaxKeyBytes,
				"%s / %s: an accepted key must produce a storable object name", label, scope)

			// The prefix a restore lists under has to fit too, or the lookup fails
			// where the save would have succeeded.
			assert.LessOrEqual(t, len(ObjectNamePrefix(name, name, scope, namespace, key)), s3MaxKeyBytes,
				"%s / %s: the restore prefix has to fit as well", label, scope)
		}
	}
}

// TestPullRequestNamespaceSeparatesRepositories is the property a pull request number
// cannot carry on its own.
//
// One workflow can be the target of triggers watching different repositories, and pull
// request numbers restart at 1 in each of them. Keying the namespace on the number alone
// put #42 over there and #42 over here in the same folder, where each could read what the
// other stored and seed what the other restores - which is the isolation the namespaces
// exist to provide, lost between two untrusted runs instead of between one and a trusted
// run.
func TestPullRequestNamespaceSeparatesRepositories(t *testing.T) {
	const (
		repoA = "https://github.com/acme/api/pull/42"
		repoB = "https://github.com/acme/web/pull/42"
		// A different host entirely, in case identity were taken from the path alone.
		repoC = "https://git.example.com/acme/api/pull/42"
	)

	a := PullRequestNamespace(repoA, "42")
	b := PullRequestNamespace(repoB, "42")
	c := PullRequestNamespace(repoC, "42")

	assert.NotEqual(t, a, b, "the same number in two repositories must not share a namespace")
	assert.NotEqual(t, a, c, "nor across hosts")
	assert.NotEqual(t, b, c)

	// The same pull request resolves the same way every run, or it would never hit its
	// own cache.
	assert.Equal(t, a, PullRequestNamespace(repoA, "42"))

	// The number stays legible, which is the whole reason it is not simply hashed away.
	assert.True(t, strings.HasPrefix(a, "pr-42-"), "got %q", a)

	t.Run("a long head ref still carries the repository", func(t *testing.T) {
		ref := strings.Repeat("feature/", 40)
		x := PullRequestNamespace(repoA, ref)
		y := PullRequestNamespace(repoB, ref)
		assert.NotEqual(t, x, y, "the digested form has to include the repository too")
		assert.LessOrEqual(t, len(x), maxNamespaceChars)
	})

	t.Run("the two forms cannot be confused", func(t *testing.T) {
		// A readable namespace and a digested one are told apart by the underscore, so
		// no identifier can be chosen to spell a namespace of the other shape.
		readable := PullRequestNamespace(repoA, "42")
		digested := PullRequestNamespace(repoA, strings.Repeat("z", 200))
		assert.False(t, strings.HasPrefix(readable, pullRequestNamespacePrefix+"_"))
		assert.True(t, strings.HasPrefix(digested, pullRequestNamespacePrefix+"_"))
	})

	t.Run("a trigger that reported no URL keeps the older shape", func(t *testing.T) {
		// Ambiguous, and unavoidably so - there is nothing to distinguish repositories
		// by. Pinned so that it is a decision rather than a regression.
		assert.Equal(t, "pr-42", PullRequestNamespace("", "42"))
	})
}

// TestResolveNamespacesUsesTheRepository covers the same property through the decision
// the control planes actually call.
func TestResolveNamespacesUsesTheRepository(t *testing.T) {
	config := func(pairs map[string]string) map[string]testkube.TestWorkflowExecutionConfigValue {
		out := make(map[string]testkube.TestWorkflowExecutionConfigValue, len(pairs))
		for k, v := range pairs {
			out[k] = testkube.TestWorkflowExecutionConfigValue{Value: v}
		}
		return out
	}

	a, readOnlyA := ResolveNamespaces(config(map[string]string{
		ConfigKeyPRNumber: "42",
		ConfigKeyPRURL:    "https://github.com/acme/api/pull/42",
	}))
	b, _ := ResolveNamespaces(config(map[string]string{
		ConfigKeyPRNumber: "42",
		ConfigKeyPRURL:    "https://github.com/acme/web/pull/42",
	}))

	assert.NotEqual(t, a, b, "two repositories, one number, two namespaces")
	assert.Equal(t, BaseNamespace, readOnlyA, "both still read the base namespace")

	// A URL on its own still marks the run as a pull request's.
	only, readOnly := ResolveNamespaces(config(map[string]string{
		ConfigKeyPRURL: "https://github.com/acme/api/pull/42",
	}))
	assert.NotEqual(t, BaseNamespace, only)
	assert.Equal(t, BaseNamespace, readOnly)

	// And a run with no git metadata at all is still trusted.
	trusted, none := ResolveNamespaces(nil)
	assert.Equal(t, BaseNamespace, trusted)
	assert.Empty(t, none)
}
