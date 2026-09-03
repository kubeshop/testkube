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

	match, exact, found := executioncache.MatchRestore(entries, scope.objectName(req.Key), prefixes)
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

	scope, err := s.resolveCacheScope(ctx, req.Id, req.Scope)
	if err != nil {
		return nil, err
	}

	objectName := scope.objectName(req.Key)

	// An entry is immutable for its lifetime. That is what makes a content-hash key
	// trustworthy: the first writer of a key wins, so a later run cannot swap out what
	// an earlier one stored, and a race between two executions costs one wasted upload
	// rather than an overwrite.
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
