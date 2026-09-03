package executioncache

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RestoreRequest asks for the entry a step should restore.
type RestoreRequest struct {
	// Key is the exact key the step computed.
	Key string
	// RestoreKeys are fallback prefixes, in the order the workflow declared them.
	RestoreKeys []string
	Scope       Scope
}

// RestoreResult is where to fetch a restorable entry from.
//
// A miss is Hit == false with a nil error, not an error: the step then installs from
// the network exactly as it did before any cache existed.
type RestoreResult struct {
	Hit bool
	// Exact reports whether the entry is the key asked for rather than a fallback.
	// A save afterwards is redundant when it is true.
	Exact      bool
	MatchedKey string
	URL        string
	Size       int64
}

// SaveRequest asks where to upload a new entry.
type SaveRequest struct {
	Key   string
	Scope Scope
	// Size is the compressed size about to be uploaded, so that a quota can be
	// refused before the transfer rather than part-way through it.
	Size int64
}

// SaveResult is where to upload to.
type SaveResult struct {
	URL string
	// AlreadyExists reports that this key is already stored, so the upload is skipped.
	//
	// This is the cheap path, not the guarantee. Looking the key up and granting the URL
	// are two steps, so two executions saving the same key can both be told it is
	// absent; what stops the second one overwriting the first is Headers below.
	AlreadyExists bool
	// Headers must be sent with the upload, verbatim.
	//
	// They carry a condition - "only if this object does not exist" - and are covered by
	// the signature, so an upload that drops them is rejected rather than quietly
	// becoming an overwrite. That is what makes an entry immutable once stored: of two
	// executions racing on one key, exactly one upload can succeed.
	//
	// The losing upload is refused by the store, which is a normal outcome and not a
	// failure: the other execution stored an equivalent tree under the same
	// content-derived key. Callers should treat a refusal the way they treat
	// AlreadyExists.
	Headers map[string]string
}

// UploadRefused reports whether an upload was rejected because the entry already
// existed, which is how a store answers a conditional write it will not apply.
//
// Kept here so both the agent and any other caller agree on which status means "someone
// else won the race" rather than "the upload failed".
func UploadRefused(statusCode int) bool {
	// 412 is the conditional-write refusal; S3 also answers 409 for a conflicting
	// concurrent write on the same key.
	return statusCode == http.StatusPreconditionFailed || statusCode == http.StatusConflict
}

// Repository is what the cache commands need from the control plane.
//
// It exists as an interface for two reasons. It keeps the commands testable without a
// control plane or an object store, and it isolates them from the transport, so the
// pod-side behaviour can be settled independently of the wire format.
type Repository interface {
	Restore(ctx context.Context, req RestoreRequest) (RestoreResult, error)
	Save(ctx context.Context, req SaveRequest) (SaveResult, error)
}

// unsupportedRepository stands in where caching cannot work at all - the control plane
// does not advertise the capability, or the pod has no connection to one.
//
// It reports a miss rather than an error, because that is the contract: a workflow
// written against a control plane that can cache must still run against one that
// cannot. The reason is carried so the caller can say why nothing was cached; a silent
// miss would leave an operator with no way to tell a cold cache from a disabled one.
type unsupportedRepository struct {
	reason string
}

// Unsupported returns a Repository that misses every restore and skips every save.
func Unsupported(reason string) Repository {
	return unsupportedRepository{reason: reason}
}

func (r unsupportedRepository) Restore(context.Context, RestoreRequest) (RestoreResult, error) {
	return RestoreResult{}, nil
}

func (r unsupportedRepository) Save(context.Context, SaveRequest) (SaveResult, error) {
	// AlreadyExists short-circuits the upload without claiming anything was stored.
	return SaveResult{AlreadyExists: true}, nil
}

// Reason explains why this repository cannot cache, for the caller's log line.
func (r unsupportedRepository) Reason() string {
	return r.reason
}

// Reason returns why a repository cannot cache, or "" when it can.
func Reason(repository Repository) string {
	if unsupported, ok := repository.(unsupportedRepository); ok {
		return unsupported.reason
	}
	return ""
}

// IsUnsupported reports whether an error means the control plane does not implement
// dependency caching.
//
// Checking the capability up front is not enough on its own: a control plane that
// forgets to withdraw the capability still has to behave comprehensibly, and one that
// has never heard of these methods answers Unimplemented. Both must land on a miss.
// This deliberately differs from pkg/executiondata, which turns the same condition into
// a hard error - a workflow that cannot read an artifact it asked for has nothing left
// to do, whereas one that cannot read a cache can simply rebuild.
func IsUnsupported(err error) bool {
	return status.Code(err) == codes.Unimplemented
}

// IsRefused reports whether an error means the control plane declined to cache this
// entry, as opposed to failing to answer. A refusal - a quota, a permission - is the
// control plane working correctly and saying no, so it degrades to a miss too.
func IsRefused(err error) bool {
	switch status.Code(err) {
	case codes.ResourceExhausted, codes.PermissionDenied, codes.InvalidArgument, codes.NotFound:
		return true
	default:
		return false
	}
}

// Degraded turns an error into the reason a cache operation was skipped, or returns the
// error unchanged when it is not something a cache may swallow.
func Degraded(err error) (reason string, degraded bool) {
	switch {
	case err == nil:
		return "", false
	case IsUnsupported(err):
		return fmt.Sprintf("this control plane cannot serve dependency caches (missing the %q capability): upgrade it to enable caching", CapabilityName), true
	case IsRefused(err):
		return err.Error(), true
	default:
		return "", false
	}
}

// CapabilityName mirrors capabilities.CapabilityDependencyCache.
//
// Held as a plain string so that this package, which the control plane also imports,
// does not drag in the capability package for one message.
const CapabilityName = "tw-cache"
