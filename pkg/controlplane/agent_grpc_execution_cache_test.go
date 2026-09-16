package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

const (
	cacheTestEnv      = "env-1"
	cacheTestWorkflow = "wf-a"
	cacheTestBucket   = "artifacts"
)

func newCacheServer(t *testing.T) (*Server, *storage.MockClient, *testworkflow.MockRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	storageClient := storage.NewMockClient(ctrl)
	repository := testworkflow.NewMockRepository(ctrl)
	return &Server{
		storageClient:     storageClient,
		resultsRepository: repository,
		cfg:               Config{StorageBucket: cacheTestBucket},
		envID:             cacheTestEnv,
	}, storageClient, repository
}

// expectExecution makes the repository answer with a running execution of the named
// workflow, which is what the handlers derive the cache scope from.
func expectExecution(repository *testworkflow.MockRepository, workflowName string) {
	repository.EXPECT().Get(gomock.Any(), "exec-1").Return(testkube.TestWorkflowExecution{
		Id:       "exec-1",
		Workflow: &testkube.TestWorkflow{Name: workflowName},
	}, nil).AnyTimes()
}

// cacheObject is where an entry for the given key belongs, computed the same way the
// agent and the commercial control plane compute it.
func cacheObject(scope executioncache.Scope, key string) string {
	return executioncache.ObjectName(cacheTestEnv, cacheTestWorkflow, scope, executioncache.BaseNamespace, key)
}

// expectExactProbe mirrors the handler's direct lookup of the exact key.
//
// It runs before the scope listing on purpose: a scope holding more entries than
// MaxCacheRestoreCandidates could otherwise hide an exact hit behind the limit, so the
// key a workflow actually asked for is never left to chance. Passing no entries makes
// the probe a miss and the scope listing below is then what answers.
func expectExactProbe(storageClient *storage.MockClient, objectName string, entries ...storage.ObjectInfo) {
	storageClient.EXPECT().
		ListObjectsFromBucket(gomock.Any(), cacheTestBucket, objectName, 1).
		Return(entries, nil)
}

// expectScopeListing mirrors the bounded listing of everything a scope can see, which
// is what the restore keys are matched against.
func expectScopeListing(storageClient *storage.MockClient, entries []storage.ObjectInfo, err error) *gomock.Call {
	return storageClient.EXPECT().
		ListObjectsFromBucket(gomock.Any(), cacheTestBucket, gomock.Any(), MaxCacheRestoreCandidates).
		Return(entries, err)
}

func TestGetExecutionCachePresigned(t *testing.T) {
	t.Run("exact hit is signed at the key that was asked for", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		wanted := cacheObject(executioncache.ScopeWorkflow, "npm-abc")
		// The probe answers, so the scope is never listed at all.
		expectExactProbe(storageClient, wanted, storage.ObjectInfo{Key: wanted, Size: 42, LastModified: time.Now()})
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, gomock.Any(), MaxCacheRestoreCandidates).
			Times(0)
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
			Return("https://storage/entry", nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.True(t, res.Hit)
		assert.True(t, res.Exact)
		assert.Equal(t, "npm-abc", res.MatchedKey, "the key is reported as the author wrote it, not encoded")
		assert.Equal(t, "https://storage/entry", res.Url)
		assert.EqualValues(t, 42, res.Size)
	})

	t.Run("restore key falls back to the newest match", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		older := cacheObject(executioncache.ScopeWorkflow, "npm-old")
		newer := cacheObject(executioncache.ScopeWorkflow, "npm-new")
		expectExactProbe(storageClient, cacheObject(executioncache.ScopeWorkflow, "npm-absent"))
		expectScopeListing(storageClient, []storage.ObjectInfo{
			{Key: older, LastModified: time.Now().Add(-time.Hour)},
			{Key: newer, LastModified: time.Now()},
		}, nil)
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), cacheTestBucket, "", newer, CachePresignedURLExpiration).
			Return("https://storage/newer", nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-absent", RestoreKeys: []string{"npm-"},
		})

		require.NoError(t, err)
		assert.True(t, res.Hit)
		assert.False(t, res.Exact, "a fallback must be reported as inexact so the step still saves its own key")
		assert.Equal(t, "npm-new", res.MatchedKey)
	})

	t.Run("an empty restore key is ignored rather than matching the whole scope", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		expectExactProbe(storageClient, cacheObject(executioncache.ScopeWorkflow, "npm-absent"))
		expectScopeListing(storageClient, []storage.ObjectInfo{
			{Key: cacheObject(executioncache.ScopeWorkflow, "unrelated"), LastModified: time.Now()},
		}, nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-absent", RestoreKeys: []string{""},
		})

		require.NoError(t, err)
		assert.False(t, res.Hit)
	})

	t.Run("a miss is an empty response, not an error", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		expectExactProbe(storageClient, cacheObject(executioncache.ScopeWorkflow, "npm-abc"))
		expectScopeListing(storageClient, nil, nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.False(t, res.Hit)
		assert.Empty(t, res.Url)
	})

	t.Run("a bucket that does not exist yet is a miss", func(t *testing.T) {
		// A cold cache is the normal first state, so it must not surface as a fault.
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		// Both lookups meet the missing bucket, and neither may surface as a fault.
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, gomock.Any(), 1).
			Return(nil, minio.ErrArtifactsNotFound)
		expectScopeListing(storageClient, nil, minio.ErrArtifactsNotFound)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.False(t, res.Hit)
	})

	// The scope is what the whole feature's isolation rests on, so assert the paths
	// rather than trusting them.
	t.Run("the environment scope reaches a different place from the workflow scope", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		var listed string
		expectExactProbe(storageClient, cacheObject(executioncache.ScopeEnvironment, "npm-abc"))
		expectScopeListing(storageClient, nil, nil).
			DoAndReturn(func(_ context.Context, _, prefix string, _ int) ([]storage.ObjectInfo, error) {
				listed = prefix
				return nil, nil
			})

		_, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id:    "exec-1",
			Key:   "npm-abc",
			Scope: cloud.ExecutionCacheScope_EXECUTION_CACHE_SCOPE_ENVIRONMENT,
		})

		require.NoError(t, err)
		assert.Contains(t, listed, "/e", "an environment-scoped lookup must not read a workflow's folder")
		assert.NotContains(t, listed, cacheTestWorkflow,
			"the workflow name must not appear in a shared scope")
	})

	t.Run("a hostile key never reaches storage", func(t *testing.T) {
		// The strongest form of the confinement claim: an over-long or empty key is
		// refused before any object name is derived from it at all.
		for _, key := range []string{"", string(make([]byte, executioncache.MaxKeyBytes+1))} {
			server, _, repository := newCacheServer(t)
			repository.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

			_, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
				Id: "exec-1", Key: key,
			})

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		}
	})

	t.Run("requires an execution id", func(t *testing.T) {
		server, _, _ := newCacheServer(t)
		_, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{Key: "k"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("an unknown execution is refused", func(t *testing.T) {
		server, _, repository := newCacheServer(t)
		repository.EXPECT().Get(gomock.Any(), "exec-1").Return(testkube.TestWorkflowExecution{}, errors.New("nope"))

		_, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("a finished execution is refused", func(t *testing.T) {
		// A cache write from a completed execution is either a bug or an attempt to
		// reach a scope the caller should no longer hold.
		server, _, repository := newCacheServer(t)
		finishedAt := time.Now()
		repository.EXPECT().Get(gomock.Any(), "exec-1").Return(testkube.TestWorkflowExecution{
			Id:       "exec-1",
			Workflow: &testkube.TestWorkflow{Name: cacheTestWorkflow},
			Result: &testkube.TestWorkflowResult{
				Status:     common.Ptr(testkube.PASSED_TestWorkflowStatus),
				FinishedAt: finishedAt,
			},
		}, nil)

		_, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

func TestSaveExecutionCachePresigned(t *testing.T) {
	t.Run("grants an upload at the derived key", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		wanted := cacheObject(executioncache.ScopeWorkflow, "npm-abc")
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, wanted, 1).
			Return(nil, nil)
		storageClient.EXPECT().
			PresignCreateFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
			Return("https://storage/put", map[string]string{"If-None-Match": "*"}, nil)

		res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc", Size: 1024,
		})

		require.NoError(t, err)
		assert.Equal(t, "https://storage/put", res.Url)
		assert.False(t, res.AlreadyExists)
	})

	// Deduplication, not immutability: this skips the upload for a key that is already
	// there, but the lookup and the grant are two steps, so it cannot stop two
	// executions racing on the same key. See SaveExecutionCachePresigned.
	t.Run("an existing key is not granted a second write", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		wanted := cacheObject(executioncache.ScopeWorkflow, "npm-abc")
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, wanted, 1).
			Return([]storage.ObjectInfo{{Key: wanted}}, nil)
		storageClient.EXPECT().PresignCreateFileToBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.True(t, res.AlreadyExists)
		assert.Empty(t, res.Url)
	})

	t.Run("a hostile key never reaches storage", func(t *testing.T) {
		server, _, repository := newCacheServer(t)
		repository.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

		_, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
			Id: "exec-1", Key: "",
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

// TestCacheObjectNamesAreConfined pins that whatever a workflow puts in a key, the
// object name the handlers derive stays inside that workflow's own folder. The
// derivation lives in pkg/executioncache and is tested there too; this asserts the
// handlers actually use it.
func TestCacheObjectNamesAreConfined(t *testing.T) {
	for _, key := range []string{"../../e/shared", "a/b", "/abs", "..", "npm-你好"} {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		scopePrefix := executioncache.ScopePrefix(cacheTestEnv, cacheTestWorkflow, executioncache.ScopeWorkflow, executioncache.BaseNamespace)
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, gomock.Any(), 1).
			DoAndReturn(func(_ context.Context, _, objectName string, _ int) ([]storage.ObjectInfo, error) {
				assert.True(t, len(objectName) > len(scopePrefix) && objectName[:len(scopePrefix)] == scopePrefix,
					"key %q escaped its scope: %s", key, objectName)
				return nil, nil
			})
		storageClient.EXPECT().
			PresignCreateFileToBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("https://storage/put", nil, nil)

		_, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
			Id: "exec-1", Key: key,
		})
		require.NoError(t, err)
	}
}

// TestSaveExecutionCachePresignedRejectsANegativeSize guards a value nothing acts on
// yet, which is the reason to reject it now: the size is what a quota would be enforced
// against, so once enforcement exists a negative one would compare below every limit.
func TestSaveExecutionCachePresignedRejectsANegativeSize(t *testing.T) {
	server, storageClient, repository := newCacheServer(t)
	// Refused before the execution is even looked up, let alone storage touched.
	repository.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
	storageClient.EXPECT().ListObjectsFromBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
		Id: "exec-1", Key: "npm-abc", Size: -1,
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "cannot be negative")
}

// TestSaveExecutionCachePresignedAcceptsAZeroSize: an empty cached directory is a
// legitimate thing to store, so zero is not the same as negative.
func TestSaveExecutionCachePresignedAcceptsAZeroSize(t *testing.T) {
	server, storageClient, repository := newCacheServer(t)
	expectExecution(repository, cacheTestWorkflow)

	wanted := cacheObject(executioncache.ScopeWorkflow, "npm-abc")
	storageClient.EXPECT().ListObjectsFromBucket(gomock.Any(), cacheTestBucket, wanted, 1).Return(nil, nil)
	storageClient.EXPECT().
		PresignCreateFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
		Return("https://storage/put", map[string]string{"If-None-Match": "*"}, nil)

	res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
		Id: "exec-1", Key: "npm-abc", Size: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage/put", res.Url)
}

// TestSaveExecutionCachePresignedReturnsTheCondition is what makes a stored entry
// immutable rather than merely deduplicated.
//
// The grant must be a conditional write and must hand the agent the headers carrying
// that condition: they are covered by the signature, so an upload that omits them is
// rejected instead of becoming a plain overwrite, and of two executions racing on one
// key exactly one upload can be applied.
func TestSaveExecutionCachePresignedReturnsTheCondition(t *testing.T) {
	server, storageClient, repository := newCacheServer(t)
	expectExecution(repository, cacheTestWorkflow)

	wanted := cacheObject(executioncache.ScopeWorkflow, "npm-abc")
	storageClient.EXPECT().ListObjectsFromBucket(gomock.Any(), cacheTestBucket, wanted, 1).Return(nil, nil)
	// The unconditional presign must not be used for a cache entry at all.
	storageClient.EXPECT().PresignUploadFileToBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	storageClient.EXPECT().
		PresignCreateFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
		Return("https://storage/put", map[string]string{"If-None-Match": "*"}, nil)

	res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
		Id: "exec-1", Key: "npm-abc", Size: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage/put", res.Url)
	assert.Equal(t, map[string]string{"If-None-Match": "*"}, res.RequiredHeaders,
		"the agent cannot satisfy a signed condition it was not told about")
}

// expectPullRequestExecution makes the repository answer with a running execution that
// the trigger recorded git metadata on, which is how a pull request's run is recognised.
//
// The config is written server-side when the execution is scheduled. That is the whole
// reason it can be trusted for this: the author of a pull request controls the code its
// run executes, so anything that travelled with the agent's request would let them pick
// the namespace they write into.
func expectPullRequestExecution(repository *testworkflow.MockRepository, prNumber string) {
	repository.EXPECT().Get(gomock.Any(), "exec-1").Return(testkube.TestWorkflowExecution{
		Id:       "exec-1",
		Workflow: &testkube.TestWorkflow{Name: cacheTestWorkflow},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{
			configKeyPRNumber: {Value: prNumber},
		},
	}, nil).AnyTimes()
}

func prCacheObject(prNumber string, scope executioncache.Scope, key string) string {
	return executioncache.ObjectName(cacheTestEnv, cacheTestWorkflow, scope,
		executioncache.PullRequestNamespace(prNumber), key)
}

// TestResolveNamespaces pins which runs are treated as a pull request's.
//
// Everything that is not one shares the base namespace - a push to any branch, a tag, a
// schedule, a manual run - so the ordinary case keeps a single cache rather than
// fragmenting into one per branch, which is what would quietly destroy the hit rate.
func TestResolveNamespaces(t *testing.T) {
	base := executioncache.BaseNamespace

	t.Run("a run with no git metadata is trusted", func(t *testing.T) {
		namespace, readOnly := resolveNamespaces(nil)
		assert.Equal(t, base, namespace)
		assert.Empty(t, readOnly, "a trusted run already writes the namespace it would fall back to")
	})

	t.Run("a push is trusted", func(t *testing.T) {
		namespace, readOnly := resolveNamespaces(map[string]testkube.TestWorkflowExecutionConfigValue{
			"TESTKUBE_GIT_BRANCH": {Value: "main"},
			"TESTKUBE_GIT_COMMIT": {Value: "abc123"},
		})
		assert.Equal(t, base, namespace)
		assert.Empty(t, readOnly)
	})

	t.Run("a pull request writes its own namespace and reads the base one", func(t *testing.T) {
		namespace, readOnly := resolveNamespaces(map[string]testkube.TestWorkflowExecutionConfigValue{
			configKeyPRNumber: {Value: "42"},
		})
		assert.Equal(t, executioncache.PullRequestNamespace("42"), namespace)
		assert.Equal(t, base, readOnly)
	})

	t.Run("the head ref identifies it when no number was reported", func(t *testing.T) {
		namespace, _ := resolveNamespaces(map[string]testkube.TestWorkflowExecutionConfigValue{
			configKeyPRHeadRef: {Value: "feature/login"},
		})
		assert.Equal(t, executioncache.PullRequestNamespace("feature/login"), namespace)
	})

	t.Run("an empty value is not a pull request", func(t *testing.T) {
		namespace, readOnly := resolveNamespaces(map[string]testkube.TestWorkflowExecutionConfigValue{
			configKeyPRNumber:  {Value: ""},
			configKeyPRHeadRef: {Value: ""},
		})
		assert.Equal(t, base, namespace)
		assert.Empty(t, readOnly)
	})
}

// TestPullRequestCannotWriteWhereATrustedRunReads is the property the namespaces exist
// for, and the only one worth breaking the layout over.
//
// A pull request's run executes code its author wrote. Before this, everything a workflow
// had ever stored sat in one namespace, so that run could store an entry which a later
// run for the default branch restored through a restoreKeys prefix - and for a build
// cache, whose contents are trusted on a key match rather than verified, that is code
// that then runs.
func TestPullRequestCannotWriteWhereATrustedRunReads(t *testing.T) {
	t.Run("a pull request's save is granted in its own namespace", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectPullRequestExecution(repository, "42")

		wanted := prCacheObject("42", executioncache.ScopeWorkflow, "npm-abc")
		require.NotEqual(t, wanted, cacheObject(executioncache.ScopeWorkflow, "npm-abc"),
			"the fixture has to differ from the base namespace or this asserts nothing")

		expectExactProbe(storageClient, wanted)
		storageClient.EXPECT().
			PresignCreateFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
			Return("https://storage/put", map[string]string{"If-None-Match": "*"}, nil)

		res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc", Size: 1024,
		})

		require.NoError(t, err)
		assert.Equal(t, "https://storage/put", res.Url)
	})

	t.Run("a trusted run never lists a pull request's namespace", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectExecution(repository, cacheTestWorkflow)

		basePrefix := executioncache.ScopePrefix(cacheTestEnv, cacheTestWorkflow,
			executioncache.ScopeWorkflow, executioncache.BaseNamespace) + "/"

		expectExactProbe(storageClient, cacheObject(executioncache.ScopeWorkflow, "npm-abc"))
		// The listing is rooted at the base namespace, so a restore key however broad
		// cannot reach an entry a pull request stored.
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, basePrefix, MaxCacheRestoreCandidates).
			Return(nil, nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc", RestoreKeys: []string{"npm-", ""},
		})

		require.NoError(t, err)
		assert.False(t, res.Hit)
	})
}

// TestPullRequestFallsBackToTheBaseNamespace covers the half that keeps the feature worth
// having: a pull request starts from what the default branch already built, rather than
// from nothing, and still cannot change what the default branch will later restore.
func TestPullRequestFallsBackToTheBaseNamespace(t *testing.T) {
	t.Run("its own namespace is consulted first", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectPullRequestExecution(repository, "42")

		own := prCacheObject("42", executioncache.ScopeWorkflow, "npm-abc")
		expectExactProbe(storageClient, own, storage.ObjectInfo{Key: own, Size: 7, LastModified: time.Now()})
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), cacheTestBucket, "", own, CachePresignedURLExpiration).
			Return("https://storage/own", nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.True(t, res.Hit)
		assert.Equal(t, "https://storage/own", res.Url, "its own entry wins over the base one, being fresher")
	})

	t.Run("the base namespace answers when its own holds nothing", func(t *testing.T) {
		server, storageClient, repository := newCacheServer(t)
		expectPullRequestExecution(repository, "42")

		own := prCacheObject("42", executioncache.ScopeWorkflow, "npm-abc")
		base := cacheObject(executioncache.ScopeWorkflow, "npm-abc")

		ownPrefix := executioncache.ScopePrefix(cacheTestEnv, cacheTestWorkflow,
			executioncache.ScopeWorkflow, executioncache.PullRequestNamespace("42")) + "/"

		// Its own namespace: nothing, either exactly or by prefix.
		expectExactProbe(storageClient, own)
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, ownPrefix, MaxCacheRestoreCandidates).
			Return(nil, nil)

		// Then the base one, which has it.
		expectExactProbe(storageClient, base, storage.ObjectInfo{Key: base, Size: 9, LastModified: time.Now()})
		storageClient.EXPECT().
			PresignDownloadFileFromBucket(gomock.Any(), cacheTestBucket, "", base, CachePresignedURLExpiration).
			Return("https://storage/base", nil)

		res, err := server.GetExecutionCachePresigned(context.Background(), &cloud.GetExecutionCachePresignedRequest{
			Id: "exec-1", Key: "npm-abc",
		})

		require.NoError(t, err)
		assert.True(t, res.Hit)
		assert.Equal(t, "https://storage/base", res.Url)
	})

	t.Run("a trusted run has nothing to fall back to", func(t *testing.T) {
		scope := cacheScope{namespace: executioncache.BaseNamespace}
		_, ok := scope.readOnly()
		assert.False(t, ok, "a trusted run already writes the namespace it would read")
	})
}
