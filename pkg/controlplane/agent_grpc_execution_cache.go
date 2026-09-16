package controlplane

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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

// Config keys the trigger records on an execution when the event carried git metadata.
// They are written server-side at schedule time, which is what makes them usable here:
// the author of a pull request controls the code its run executes, so anything that
// travelled with the request would let them choose the namespace they write into.
const (
	configKeyPRNumber  = "TESTKUBE_GIT_PR_NUMBER"
	configKeyPRHeadRef = "TESTKUBE_GIT_PR_HEAD_REF"
)

// resolveNamespaces decides which namespace an execution writes and which, if any, it may
// additionally read.
//
// A run is a pull request's when the trigger said so. Everything else - a push to any
// branch, a tag, a schedule, a manual run - is trusted and shares the base namespace, so
// the ordinary case keeps one cache rather than one per branch.
//
// The pull request number identifies the namespace where it exists, because it survives a
// force-push and a rename of the head branch. The head ref is the fallback for a trigger
// that reported one without the other.
func resolveNamespaces(config map[string]testkube.TestWorkflowExecutionConfigValue) (namespace, readOnly string) {
	identifier := ""
	for _, key := range []string{configKeyPRNumber, configKeyPRHeadRef} {
		if value, ok := config[key]; ok && value.Value != "" {
			identifier = value.Value
			break
		}
	}
	if identifier == "" {
		return executioncache.BaseNamespace, ""
	}
	return executioncache.PullRequestNamespace(identifier), executioncache.BaseNamespace
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

	namespace, readOnly := resolveNamespaces(execution.ConfigParams)

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

	entries, err := s.listCacheEntries(ctx, scope.prefix()+"/", MaxCacheRestoreCandidates)
	if err != nil {
		return nil, err
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

	match, exact, found := executioncache.MatchRestore(entries, exactObject, prefixes)
	if !found {
		return nil, nil
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
