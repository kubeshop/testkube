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
	return executioncache.ObjectName(cacheTestEnv, cacheTestWorkflow, scope, key)
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
			PresignUploadFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
			Return("https://storage/put", nil)

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
		storageClient.EXPECT().PresignUploadFileToBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

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

		scopePrefix := executioncache.ScopePrefix(cacheTestEnv, cacheTestWorkflow, executioncache.ScopeWorkflow)
		storageClient.EXPECT().
			ListObjectsFromBucket(gomock.Any(), cacheTestBucket, gomock.Any(), 1).
			DoAndReturn(func(_ context.Context, _, objectName string, _ int) ([]storage.ObjectInfo, error) {
				assert.True(t, len(objectName) > len(scopePrefix) && objectName[:len(scopePrefix)] == scopePrefix,
					"key %q escaped its scope: %s", key, objectName)
				return nil, nil
			})
		storageClient.EXPECT().
			PresignUploadFileToBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("https://storage/put", nil)

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
		PresignUploadFileToBucket(gomock.Any(), cacheTestBucket, "", wanted, CachePresignedURLExpiration).
		Return("https://storage/put", nil)

	res, err := server.SaveExecutionCachePresigned(context.Background(), &cloud.SaveExecutionCachePresignedRequest{
		Id: "exec-1", Key: "npm-abc", Size: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, "https://storage/put", res.Url)
}
