package controlplaneclient

import (
	"context"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executioncache"
)

// ExecutionCacheClient reads and writes a step's dependency cache.
//
// Kept apart from ExecutionSelfClient rather than added to it, because that interface is
// the narrow type the artifact uploader takes; widening it would make the uploader
// depend on cache methods it never calls.
type ExecutionCacheClient interface {
	// GetExecutionCachePresignedURL asks whether an entry exists for this execution's
	// workflow, and for a URL to download it from. A miss is not an error.
	GetExecutionCachePresignedURL(ctx context.Context, environmentId, executionId string, req executioncache.RestoreRequest) (executioncache.RestoreResult, error)
	// SaveExecutionCacheGetPresignedURL asks for a URL to upload a new entry to.
	SaveExecutionCacheGetPresignedURL(ctx context.Context, environmentId, executionId string, req executioncache.SaveRequest) (executioncache.SaveResult, error)
}

// cacheScopeToProto maps the scope onto the wire.
//
// The default arm is deliberate: an unrecognised scope becomes the narrowest one rather
// than the widest, so a value this layer fails to understand cannot widen who is able to
// write what a workflow will later execute.
func cacheScopeToProto(scope executioncache.Scope) cloud.ExecutionCacheScope {
	if scope == executioncache.ScopeEnvironment {
		return cloud.ExecutionCacheScope_EXECUTION_CACHE_SCOPE_ENVIRONMENT
	}
	return cloud.ExecutionCacheScope_EXECUTION_CACHE_SCOPE_WORKFLOW
}

func (c *client) GetExecutionCachePresignedURL(ctx context.Context, environmentId, executionId string, req executioncache.RestoreRequest) (executioncache.RestoreResult, error) {
	request := cloud.GetExecutionCachePresignedRequest{
		Id:          executionId,
		Key:         req.Key,
		RestoreKeys: req.RestoreKeys,
		Scope:       cacheScopeToProto(req.Scope),
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetExecutionCachePresigned, &request)
	if err != nil {
		return executioncache.RestoreResult{}, err
	}
	return executioncache.RestoreResult{
		Hit:        res.Hit,
		Exact:      res.Exact,
		MatchedKey: res.MatchedKey,
		URL:        res.Url,
		Size:       res.Size,
	}, nil
}

func (c *client) SaveExecutionCacheGetPresignedURL(ctx context.Context, environmentId, executionId string, req executioncache.SaveRequest) (executioncache.SaveResult, error) {
	request := cloud.SaveExecutionCachePresignedRequest{
		Id:    executionId,
		Key:   req.Key,
		Scope: cacheScopeToProto(req.Scope),
		Size:  req.Size,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.SaveExecutionCachePresigned, &request)
	if err != nil {
		return executioncache.SaveResult{}, err
	}
	return executioncache.SaveResult{
		URL:           res.Url,
		AlreadyExists: res.AlreadyExists,
		Headers:       res.RequiredHeaders,
	}, nil
}
