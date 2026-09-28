package controlplane

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

const (
	// CachePresignedURLExpiration is longer than the artifact grant on purpose: a
	// dependency cache can be gigabytes, and the agent already allows 30 minutes per
	// transfer attempt across several attempts.
	CachePresignedURLExpiration = 60 * time.Minute
)

// cacheScope is a resolved sharing scope: which environment and workflow an execution's
// cache entries belong to, and how widely they are shared.
type cacheScope struct {
	environmentID string
	workflowName  string
	scope         executioncache.Scope

	// namespace is the one this execution writes. For a pull request it is its own; for
	// every trusted run it is the base one.
	namespace string
	// readOnlyNamespace is consulted when namespace holds nothing, and is never written.
	// It is empty for a trusted run, which already writes the namespace it would fall
	// back to.
	readOnlyNamespace string
}

func (c cacheScope) prefix() string {
	return executioncache.ScopePrefix(c.environmentID, c.workflowName, c.scope, c.namespace)
}

func (c cacheScope) objectName(key string) string {
	return executioncache.ObjectName(c.environmentID, c.workflowName, c.scope, c.namespace, key)
}

func (c cacheScope) objectNamePrefix(keyPrefix string) string {
	return executioncache.ObjectNamePrefix(c.environmentID, c.workflowName, c.scope, c.namespace, keyPrefix)
}

// readOnly returns the scope a restore falls back to, and whether there is one.
func (c cacheScope) readOnly() (cacheScope, bool) {
	if c.readOnlyNamespace == "" || c.readOnlyNamespace == c.namespace {
		return cacheScope{}, false
	}
	fallback := c
	fallback.namespace = c.readOnlyNamespace
	fallback.readOnlyNamespace = ""
	return fallback, true
}

// resolveCacheScope decides where a request is allowed to read and write.
//
// Only the scope *kind* comes from the request. Which workflow it belongs to is read
// from the stored execution, and that is the whole reason a workflow-scoped entry cannot
// be addressed by another workflow however the request is shaped. The namespace comes
// from the same place and for the same reason: a pull request's run executes code its
// author wrote, so it must not be able to name the namespace it writes. A finished
// execution is refused: a cache write from one is either a bug or an attempt to reach a
// scope the caller should no longer hold.
func (s *Server) resolveCacheScope(ctx context.Context, executionID string, requested cloud.ExecutionCacheScope) (cacheScope, error) {
	execution, err := s.resultsRepository.Get(ctx, executionID)
	if err != nil {
		return cacheScope{}, status.Error(codes.NotFound, "execution not found")
	}
	if execution.Result != nil && execution.Result.IsFinished() {
		return cacheScope{}, status.Error(codes.FailedPrecondition, "execution is already finished")
	}
	if execution.Workflow == nil || execution.Workflow.Name == "" {
		return cacheScope{}, status.Error(codes.FailedPrecondition, "execution has no workflow")
	}

	scope := executioncache.ScopeWorkflow
	if requested == cloud.ExecutionCacheScope_EXECUTION_CACHE_SCOPE_ENVIRONMENT {
		scope = executioncache.ScopeEnvironment
	}

	namespace, readOnly := executioncache.ResolveNamespaces(execution.ConfigParams)

	return cacheScope{
		environmentID:     s.envID,
		workflowName:      execution.Workflow.Name,
		scope:             scope,
		namespace:         namespace,
		readOnlyNamespace: readOnly,
	}, nil
}

// listCacheEntries reads a scope.
//
// A scope with nothing in it surfaces as an empty listing rather than an error, matching
// how ListExecutionArtifactsPresigned treats a missing bucket: a cold cache is the normal
// first state, not a fault.
func (s *Server) listCacheEntries(ctx context.Context, prefix string, limit int) ([]executioncache.Entry, error) {
	objects, err := s.storageClient.ListObjectsFromBucket(ctx, s.cfg.StorageBucket, prefix, limit)
	if err != nil {
		if errors.Is(err, minio.ErrArtifactsNotFound) {
			return nil, nil
		}
		return nil, err
	}

	entries := make([]executioncache.Entry, 0, len(objects))
	for _, object := range objects {
		entries = append(entries, executioncache.Entry{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		})
	}
	return entries, nil
}

// streamCacheEntries hands every entry under a prefix to visit, without holding them.
//
// The same treatment of a missing bucket as listCacheEntries, and for the same reason: a
// cold cache is the normal first state, not a fault.
func (s *Server) streamCacheEntries(ctx context.Context, prefix string, visit func(executioncache.Entry) bool) error {
	// Returned by visit to end the listing, and swallowed here: stopping early is the
	// caller having read enough, not a failure of the store.
	errEnough := errors.New("enough")

	err := s.storageClient.StreamObjectsFromBucket(ctx, s.cfg.StorageBucket, prefix, func(object storage.ObjectInfo) error {
		if !visit(executioncache.Entry{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		}) {
			return errEnough
		}
		return nil
	})
	if errors.Is(err, errEnough) || errors.Is(err, minio.ErrArtifactsNotFound) {
		return nil
	}
	return err
}

// GetExecutionCachePresigned grants read access to a step's dependency cache.
func (s *Server) GetExecutionCachePresigned(ctx context.Context, req *cloud.GetExecutionCachePresignedRequest) (*cloud.GetExecutionCachePresignedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "execution id is required")
	}
	if err := executioncache.ValidateKey(req.Key); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Every restore key costs a listing of the store, and the two lookups below double
	// that, so the request cannot be allowed to name as many as it likes. The bound is
	// the one the schema already sets on the workflow; refused here, before the
	// execution is even loaded, so an over-long list costs nothing to turn away.
	if len(req.RestoreKeys) > executioncache.MaxRestoreKeys {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d restore keys are allowed, got %d",
			executioncache.MaxRestoreKeys, len(req.RestoreKeys))
	}

	scope, err := s.resolveCacheScope(ctx, req.Id, req.Scope)
	if err != nil {
		return nil, err
	}

	// The execution's own namespace first. For a trusted run that is the only one there
	// is; for a pull request it holds whatever its own earlier runs stored, which is
	// fresher than anything the base namespace could offer.
	response, err := s.lookupCacheEntry(ctx, scope, req.Key, req.RestoreKeys)
	if err != nil {
		return nil, err
	}

	// Then the one it may read but not write. This is what keeps a pull request useful:
	// it starts from the dependencies the default branch already has, without being able
	// to change what the default branch will later restore.
	if response == nil {
		if fallback, ok := scope.readOnly(); ok {
			response, err = s.lookupCacheEntry(ctx, fallback, req.Key, req.RestoreKeys)
			if err != nil {
				return nil, err
			}
		}
	}

	if response == nil {
		// A miss is a normal answer, not an error: the agent then installs from the
		// network exactly as it would without any cache.
		return &cloud.GetExecutionCachePresignedResponse{}, nil
	}
	return response, nil
}

// lookupCacheEntry answers a restore within one namespace, returning nil for a miss.
//
// Confined to the namespace it is given, which is what makes the fallback safe to run:
// the prefixes it builds are rooted at that namespace, so a restore key cannot reach
// entries belonging to anyone else even when the query is deliberately broad.
func (s *Server) lookupCacheEntry(ctx context.Context, scope cacheScope, key string, restoreKeys []string) (*cloud.GetExecutionCachePresignedResponse, error) {
	exactObject := scope.objectName(key)

	// Check the exact key directly so a large scope cannot hide an exact hit behind the listing limit.
	exactEntries, err := s.listCacheEntries(ctx, exactObject, 1)
	if err != nil {
		return nil, err
	}
	for _, entry := range exactEntries {
		if entry.Key != exactObject {
			continue
		}
		url, err := s.storageClient.PresignDownloadFileFromBucket(ctx, s.cfg.StorageBucket, "", entry.Key, CachePresignedURLExpiration)
		if err != nil {
			return nil, err
		}
		return &cloud.GetExecutionCachePresignedResponse{
			Hit:        true,
			Exact:      true,
			MatchedKey: key,
			Url:        url,
			Size:       entry.Size,
		}, nil
	}

	prefixes := make([]string, 0, len(restoreKeys))
	for _, restoreKey := range restoreKeys {
		if restoreKey == "" {
			// Would match the whole scope, which no workflow meant to ask for - and
			// under an environment scope that is another team's cache.
			continue
		}
		prefixes = append(prefixes, scope.objectNamePrefix(restoreKey))
	}
	if len(prefixes) == 0 {
		// Nothing left to try, so there is no reason to list anything at all.
		return nil, nil
	}

	// Streamed under each restore key, in the order the workflow declared them, keeping
	// only the best candidate for each.
	//
	// No page decides the answer. A store lists lexically while this policy selects on
	// recency, so a page could hide the newest entry behind entries that merely sort
	// before it, and the restore would resolve to an older dependency set rather than
	// miss. Entries are immutable, leave only on expiry, and have no cardinality limit.
	//
	// The reading is still bounded, by MaxRestoreCandidates - one lookup must not be
	// able to scan everything under a prefix. Reaching that bound produces a miss rather
	// than the best of what was read, because the best of a truncated run is not the
	// best of the prefix: a miss costs a reinstall, a wrong hit costs a build against
	// dependencies nobody asked for, repeated until the entry expires.
	//
	// The first restore key with a match wins even if a later one holds a newer entry,
	// because the order is the author stating which fallback they prefer. That is why
	// each is streamed separately rather than all at once: the loop stops at the first
	// one that matched, and never reads the rest.
	var match executioncache.Entry
	for _, prefix := range prefixes {
		candidate := executioncache.NewRestoreCandidate(prefix)
		if err := s.streamCacheEntries(ctx, prefix, candidate.Consider); err != nil {
			return nil, err
		}
		if candidate.Overflowed() {
			// Said out loud, because it is not an empty prefix and the workflow's author
			// is the only one who can narrow the key.
			s.cfg.Logger.Warnw("cache restore key matches more entries than can be chosen from, treating it as a miss",
				"prefix", prefix, "limit", executioncache.MaxRestoreCandidates)
		}
		if best, ok := candidate.Best(); ok {
			match = best
			break
		}
	}
	if match.Key == "" {
		return nil, nil
	}
	exact := match.Key == exactObject

	// The folder is left empty and the whole object name passed as the file, so the key
	// that was matched is exactly the key that gets signed.
	url, err := s.storageClient.PresignDownloadFileFromBucket(ctx, s.cfg.StorageBucket, "", match.Key, CachePresignedURLExpiration)
	if err != nil {
		return nil, err
	}

	return &cloud.GetExecutionCachePresignedResponse{
		Hit:        true,
		Exact:      exact,
		MatchedKey: executioncache.KeyFromObjectName(scope.prefix(), match.Key),
		Url:        url,
		Size:       match.Size,
	}, nil
}

// SaveExecutionCachePresigned grants write access for a new cache entry.
func (s *Server) SaveExecutionCachePresigned(ctx context.Context, req *cloud.SaveExecutionCachePresignedRequest) (*cloud.SaveExecutionCachePresignedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "execution id is required")
	}
	if err := executioncache.ValidateKey(req.Key); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// The size is what a quota would be enforced against, so a negative one is refused
	// rather than carried forward. Nothing here acts on it yet, which is precisely why
	// it is worth rejecting now: once enforcement exists, a value that compares below
	// every limit would pass it, and the agent has no reason to send one.
	if req.Size < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cache entry size cannot be negative, %d provided", req.Size)
	}

	scope, err := s.resolveCacheScope(ctx, req.Id, req.Scope)
	if err != nil {
		return nil, err
	}

	objectName := scope.objectName(req.Key)

	// Skip the upload when the key is already stored. This is only the cheap path: the
	// lookup and the grant are two steps, so two executions saving the same key can
	// both be told it is absent.
	existing, err := s.listCacheEntries(ctx, objectName, 1)
	if err != nil {
		return nil, err
	}
	for _, entry := range existing {
		if entry.Key == objectName {
			return &cloud.SaveExecutionCachePresignedResponse{AlreadyExists: true}, nil
		}
	}

	// What actually makes an entry immutable is the condition on the write. The headers
	// carrying it are signed in, so an upload that drops them is rejected instead of
	// becoming an overwrite, and of two executions racing on one key exactly one upload
	// can succeed. The loser is refused and told, rather than replacing what the winner
	// stored.
	url, headers, err := s.storageClient.PresignCreateFileToBucket(ctx, s.cfg.StorageBucket, "", objectName, CachePresignedURLExpiration)
	if err != nil {
		return nil, err
	}

	return &cloud.SaveExecutionCachePresignedResponse{Url: url, RequiredHeaders: headers}, nil
}
