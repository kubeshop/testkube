// Package executioncache derives where a step's dependency cache lives in object
// storage, and picks which stored entry a restore should use.
//
// Entries are separated by namespace as well as by scope. A run for a pull request writes
// only into its own namespace and may read the base one; every trusted run - a push, a
// tag, a schedule, a manual run - reads and writes the base namespace. Without that, a
// pull request could store an entry that a later run for the default branch restored by
// way of a restoreKeys prefix, and for a build cache that is code that then runs. The
// author of a pull request controls what executes in its run, so the identity of the
// namespace has to be derived from what the trigger recorded, never from the request.
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

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
	// The budget is what remains of 1024 after the longest layout this plane produces.
	// Counted rather than estimated, because getting it wrong produces keys that
	// validate and then fail to store:
	//
	//	.tkcache/v1   11      the fixed prefix
	//	/<env>        1 + 253 a Kubernetes name
	//	/testworkflows 1 + 13  the wider of the two scope segments
	//	/<workflow>   1 + 253 a Kubernetes name
	//	/<namespace>  1 + 68  see maxNamespaceChars
	//	/<key>        1 + ?
	//	.tar.gz       7
	//
	// That fixed part is 610, leaving 414. Rounded down to leave the commercial plane,
	// which nests deeper, a little room to share this constant.
	//
	// TestObjectNameFitsTheStore holds this to the arithmetic. It exists because the
	// namespace segment was added after this budget was first written and the budget was
	// not revisited, which put the worst case 254 bytes over the limit.
	MaxEncodedKeyBytes = 400

	// MaxKeyBytes caps the key before encoding as well, so an absurd input is refused
	// before any work is done on it. It is the same number because a key of unreserved
	// characters encodes one to one: that is the case where the raw length is the
	// encoded length, and no raw key shorter than this can be under the encoded cap by
	// virtue of its length alone.
	MaxKeyBytes = MaxEncodedKeyBytes

	// maxSegmentChars bounds a name used verbatim as a path segment. A Kubernetes name
	// cannot exceed 253 characters, so this only ever rejects something that was not one.
	maxSegmentChars = 253

	// maxNamespaceIdentifierChars bounds what a namespace may carry verbatim, beyond
	// which it is hashed instead.
	//
	// Tighter than maxSegmentChars because the identifier is not a Kubernetes name: it is
	// a pull request number, or failing that a head ref, which is a branch name of
	// whatever length its author chose. Every byte of it is a byte the cache key cannot
	// use, and a branch name long enough to matter is one nobody reads in a bucket
	// listing anyway - which is the only thing keeping it verbatim buys.
	maxNamespaceIdentifierChars = 64

	// maxReadableIdentifierChars is how much of an identifier survives verbatim when it
	// is paired with a repository digest. A pull request number is far inside it; a head
	// ref this long is hashed along with the repository instead.
	maxReadableIdentifierChars = 32

	// shortDigestChars is how much of a digest stands in for a repository. 64 bits, on a
	// value that is not attacker-chosen in any useful way: forging a collision would buy
	// a share of another repository's pull request namespace, which is another untrusted
	// namespace, and the trusted one is not reachable this way at all.
	shortDigestChars = 16

	// maxNamespaceChars is what the segment can then come to, over both forms:
	//
	//	"pr-" + 32 + "-" + 16          = 52
	//	"pr-" + "_" + 64 hex           = 68
	maxNamespaceChars = len(pullRequestNamespacePrefix) + 1 + 64

	// workflowScopeSegment and sharedScopeSegment name the two scopes in an object name.
	//
	// Spelled out rather than abbreviated, and spelled the same way the commercial
	// control plane spells them, so that an operator looking into either bucket meets
	// one vocabulary instead of two. The layouts around them stay different on purpose -
	// that plane is multi-tenant and keys by organization - but there is no reason for
	// the words to differ as well.
	workflowScopeSegment = "testworkflows"
	sharedScopeSegment   = "shared"

	// BaseNamespace holds the entries every trusted run shares - a push to any branch, a
	// tag, a schedule, a manual run. It is the only namespace that is written by a run
	// whose code was not proposed by someone outside the repository.
	BaseNamespace = "base"

	// pullRequestNamespacePrefix marks a namespace belonging to one pull request.
	//
	// It is a distinct segment rather than a bare identifier so that a restore key in the
	// base namespace cannot reach into a pull request's: the query is confined to
	// ".../base/<encoded prefix>", and a pull request's entries are not under it. Before
	// the namespaces existed, one prefix query covered every entry the workflow had ever
	// stored, whoever ran it.
	pullRequestNamespacePrefix = "pr-"
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
	return digestSegment(name)
}

// digestSegment is the fallback form: an underscore, which no Kubernetes name may begin
// with, and a full digest. That prefix is what keeps a real name from being mistaken for
// a fallback, or colliding with one.
func digestSegment(name string) string {
	sum := sha256.Sum256([]byte(name))
	return "_" + hex.EncodeToString(sum[:])
}

// shortDigest identifies a value in a segment that stays readable beside it. It carries
// no underscore, so it cannot turn a readable namespace into something a digested one
// could also spell.
func shortDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:shortDigestChars]
}

// isSafeSegment reports whether a name can be used as a path segment as it stands.
//
// An allowlist, rather than a list of forbidden characters. A Kubernetes name is already
// within it, so the readable case is unaffected; anything else - a branch name, which may
// carry spaces, slashes or any other UTF-8 - is hashed instead of being copied into an
// object name whose shape the rest of the layout relies on.
func isSafeSegment(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > maxSegmentChars {
		return false
	}
	if strings.HasPrefix(name, "_") {
		// Reserved for the fallback, so the two can never be confused.
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '-':
		default:
			return false
		}
	}
	return true
}

// PullRequestNamespace is the namespace belonging to one pull request.
//
// The identifier must come from what the trigger recorded on the execution, never from
// the request. A pull request's author controls the code that runs, so anything they
// could choose here would let them pick the namespace they write into - including the
// base one, which is the whole thing this separates them from.
// The repository has to be part of it. A pull request number is unique only within the
// repository that issued it, and one workflow can be the target of triggers watching
// different repositories - so without this, #42 over there and #42 over here share a
// namespace, and each can read and seed what the other restores. That is the isolation
// this whole split exists to provide, so it cannot rest on a number alone.
//
// A long identifier is hashed rather than carried, so that the segment stays inside
// maxNamespaceChars and the key budget above holds whatever a branch is called. The two
// forms are:
//
//	pr-42-1b833a2f4c9d5e60          a short, legible identifier and the repository digest
//	pr-_<64 hex of repository+id>   anything else, in one digest
//
// Both are bounded by maxNamespaceChars, and neither can be produced by the other's
// input: the first has no underscore after the prefix, which the second always does.
func PullRequestNamespace(repository, identifier string) string {
	if repository == "" {
		// Nothing to distinguish repositories by, so this is the older, ambiguous shape.
		// It is reachable only from a trigger that reported a pull request without its
		// URL; the collision above is then unavoidable rather than overlooked.
		if len(identifier) > maxNamespaceIdentifierChars {
			return pullRequestNamespacePrefix + digestSegment(identifier)
		}
		return pullRequestNamespacePrefix + sanitizeSegment(identifier)
	}

	if len(identifier) <= maxReadableIdentifierChars && isSafeSegment(identifier) {
		return pullRequestNamespacePrefix + identifier + "-" + shortDigest(repository)
	}
	// The separator keeps the pair unambiguous: no repository and identifier can be
	// rearranged into another pair with the same digest.
	return pullRequestNamespacePrefix + digestSegment(repository+"\n"+identifier)
}

// Config keys a trigger records on an execution when the event carried git metadata. The
// names belong to pkg/git/informer; they are restated rather than imported because that
// package pulls in go-git and a Kubernetes client, and because the commercial control
// plane has to read the same two keys without taking on either.
const (
	ConfigKeyPRNumber  = "TESTKUBE_GIT_PR_NUMBER"
	ConfigKeyPRHeadRef = "TESTKUBE_GIT_PR_HEAD_REF"
	// ConfigKeyPRURL is the only one of these that says which repository the pull
	// request belongs to, which is why the namespace is derived from it as well.
	ConfigKeyPRURL = "TESTKUBE_GIT_PR_URL"
)

// ResolveNamespaces decides which namespace an execution writes and which, if any, it may
// additionally read.
//
// A run belongs to a pull request when the trigger said so. Everything else - a push to
// any branch, a tag, a schedule, a manual run - is trusted and shares the base namespace,
// so the ordinary case keeps one cache rather than fragmenting into one per branch, which
// is what would quietly destroy the hit rate.
//
// The config must be the one stored on the execution, written server-side when it was
// scheduled. The author of a pull request controls the code its run executes, so anything
// travelling with the agent's request would let them choose the namespace they write into.
// Note which way the default falls: no marker means the base namespace, so a forged marker
// can only move a run into a weaker namespace, never into the trusted one.
//
// The pull request number identifies the namespace where there is one, because it survives
// a force-push and a rename of the head branch. The head ref is the fallback for a trigger
// that reported one without the other.
//
// Shared with the commercial control plane rather than restated there: the two planes keep
// different object layouts on purpose, but a disagreement about which runs are trusted
// would be a security difference, not a cosmetic one.
func ResolveNamespaces(config map[string]testkube.TestWorkflowExecutionConfigValue) (namespace, readOnly string) {
	valueOf := func(key string) string {
		if v, ok := config[key]; ok {
			return v.Value
		}
		return ""
	}

	identifier := valueOf(ConfigKeyPRNumber)
	if identifier == "" {
		identifier = valueOf(ConfigKeyPRHeadRef)
	}
	// Which repository it belongs to, without which a number means nothing.
	repository := valueOf(ConfigKeyPRURL)

	// The URL alone is enough to mark a run as a pull request's, for a trigger that
	// reported it without either of the others.
	if identifier == "" && repository == "" {
		return BaseNamespace, ""
	}
	return PullRequestNamespace(repository, identifier), BaseNamespace
}

// ScopePrefix is the folder holding every entry a given scope and namespace can see.
//
// Every argument must come from the execution the caller has already authenticated,
// never from the request: deriving the workflow segment server-side is the whole reason
// a workflow-scoped entry cannot be read or written by another workflow, and deriving the
// namespace server-side is the reason a pull request cannot write one a trusted run reads.
func ScopePrefix(environmentID, workflowName string, scope Scope, namespace string) string {
	env := sanitizeSegment(environmentID)
	if namespace == "" {
		namespace = BaseNamespace
	}
	if scope == ScopeEnvironment {
		return fmt.Sprintf("%s/%s/%s/%s", ObjectPrefix, env, sharedScopeSegment, namespace)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		ObjectPrefix, env, workflowScopeSegment, sanitizeSegment(workflowName), namespace)
}

// ObjectName is the object holding the entry for an exact key.
func ObjectName(environmentID, workflowName string, scope Scope, namespace, key string) string {
	return ScopePrefix(environmentID, workflowName, scope, namespace) + "/" + EncodeKey(key) + ObjectSuffix
}

// ObjectNamePrefix is the prefix matching every entry whose key starts with keyPrefix.
//
// Deliberately has no ObjectSuffix: a restore key matches on the start of a key, so the
// query has to stay open-ended.
func ObjectNamePrefix(environmentID, workflowName string, scope Scope, namespace, keyPrefix string) string {
	return ScopePrefix(environmentID, workflowName, scope, namespace) + "/" + EncodeKey(keyPrefix)
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
