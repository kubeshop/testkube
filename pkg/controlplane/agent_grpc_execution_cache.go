package controlplane

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

const (
	// CachePresignedURLExpiration is longer than the artifact grant on purpose: a
	// dependency cache can be gigabytes, and the agent already allows 30 minutes per
	// transfer attempt across several attempts.
	CachePresignedURLExpiration = 60 * time.Minute

	// MaxCacheRestoreCandidates bounds the listing behind a restore key, so a scope that
	// has accumulated many entries cannot turn one lookup into an unbounded scan.
	MaxCacheRestoreCandidates = 1000
)

// cacheScope is a resolved sharing scope: which environment and workflow an execution's
// cache entries belong to, and how widely they are shared.
type cacheScope struct {
	environmentID string
	workflowName  string
	scope         executioncache.Scope
}

func (c cacheScope) prefix() string {
	return executioncache.ScopePrefix(c.environmentID, c.workflowName, c.scope)
}

func (c cacheScope) objectName(key string) string {
	return executioncache.ObjectName(c.environmentID, c.workflowName, c.scope, key)
}

func (c cacheScope) objectNamePrefix(keyPrefix string) string {
	return executioncache.ObjectNamePrefix(c.environmentID, c.workflowName, c.scope, keyPrefix)
}

// resolveCacheScope decides where a request is allowed to read and write.
//
// Only the scope *kind* comes from the request. Which workflow it belongs to is read
// from the stored execution, and that is the whole reason a workflow-scoped entry cannot
// be addressed by another workflow however the request is shaped. A finished execution
// is refused: a cache write from one is either a bug or an attempt to reach a scope the
// caller should no longer hold.
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

	return cacheScope{
		environmentID: s.envID,
		workflowName:  execution.Workflow.Name,
		scope:         scope,
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

// GetExecutionCachePresigned grants read access to a step's dependency cache.
func (s *Server) GetExecutionCachePresigned(ctx context.Context, req *cloud.GetExecutionCachePresignedRequest) (*cloud.GetExecutionCachePresignedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "execution id is required")
	}
	if err := executioncache.ValidateKey(req.Key); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	scope, err := s.resolveCacheScope(ctx, req.Id, req.Scope)
	if err != nil {
		return nil, err
	}

	exactObject := scope.objectName(req.Key)

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
			MatchedKey: req.Key,
			Url:        url,
			Size:       entry.Size,
		}, nil
	}

	entries, err := s.listCacheEntries(ctx, scope.prefix()+"/", MaxCacheRestoreCandidates)
	if err != nil {
		return nil, err
	}

	prefixes := make([]string, 0, len(req.RestoreKeys))
	for _, restoreKey := range req.RestoreKeys {
		if restoreKey == "" {
			// Would match the whole scope, which no workflow meant to ask for - and
			// under an environment scope that is another team's cache.
			continue
		}
		prefixes = append(prefixes, scope.objectNamePrefix(restoreKey))
	}

	match, exact, found := executioncache.MatchRestore(entries, exactObject, prefixes)
	if !found {
		// A miss is a normal answer, not an error: the agent then installs from the
		// network exactly as it would without any cache.
		return &cloud.GetExecutionCachePresignedResponse{}, nil
	}

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

	// Skip the upload when the key is already stored. This is deduplication, not a
	// guarantee: the lookup and the grant are two steps, so two executions saving the
	// same key can both find nothing, both be granted a URL, and the later upload then
	// replaces the earlier entry. Making it a guarantee needs the write itself to be
	// conditional - a presigned PUT carrying If-None-Match, or a reservation the grant
	// takes atomically - neither of which this storage interface can express today.
	//
	// What that costs is bounded. Racing writers arrived at the same key, which is
	// derived from content, so they are storing equivalent trees; and a save only
	// happens after a miss, so once an entry exists later runs hit it and never write.
	// The exposure is that a workflow able to write a scope can replace an entry there
	// by racing, on top of being able to write one in the first place - which is the
	// caveat scope: environment already carries.
	existing, err := s.listCacheEntries(ctx, objectName, 1)
	if err != nil {
		return nil, err
	}
	for _, entry := range existing {
		if entry.Key == objectName {
			return &cloud.SaveExecutionCachePresignedResponse{AlreadyExists: true}, nil
		}
	}

	url, err := s.storageClient.PresignUploadFileToBucket(ctx, s.cfg.StorageBucket, "", objectName, CachePresignedURLExpiration)
	if err != nil {
		return nil, err
	}

	return &cloud.SaveExecutionCachePresignedResponse{Url: url}, nil
}
