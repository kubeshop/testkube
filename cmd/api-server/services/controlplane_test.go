package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/internal/config"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
)

// buildBuckets replicates the bucket list construction from CreateControlPlane so we
// can test it in isolation without the full k8s / database dependencies.
func buildBuckets(cfg *config.Config) []bucketSpec {
	var buckets []bucketSpec
	if cfg.ArtifactsStorage != "none" {
		buckets = append(buckets, bucketSpec{name: cfg.StorageBucket, label: "storage"})
	}
	if cfg.LogsStorage != "none" {
		buckets = append(buckets, bucketSpec{name: cfg.LogsBucket, label: "logs"})
	}
	return buckets
}

// TestBuildBuckets_BothNone confirms that when both ARTIFACTS_STORAGE=none and
// LOGS_STORAGE=none are set, no bucket specs are produced.
func TestBuildBuckets_BothNone(t *testing.T) {
	cfg := &config.Config{}
	cfg.ArtifactsStorage = "none"
	cfg.LogsStorage = "none"
	cfg.StorageBucket = "testkube-artifacts"
	cfg.LogsBucket = "testkube-logs"

	buckets := buildBuckets(cfg)
	assert.Empty(t, buckets, "expected no bucket specs when both storages are disabled")
}

// TestBuildBuckets_ArtifactsNone confirms that only the logs bucket is included
// when ARTIFACTS_STORAGE=none.
func TestBuildBuckets_ArtifactsNone(t *testing.T) {
	cfg := &config.Config{}
	cfg.ArtifactsStorage = "none"
	cfg.StorageBucket = "testkube-artifacts"
	cfg.LogsBucket = "testkube-logs"

	buckets := buildBuckets(cfg)
	assert.Len(t, buckets, 1)
	assert.Equal(t, "testkube-logs", buckets[0].name)
	assert.Equal(t, "logs", buckets[0].label)
}

// TestBuildBuckets_LogsNone confirms that only the storage bucket is included
// when LOGS_STORAGE=none.
func TestBuildBuckets_LogsNone(t *testing.T) {
	cfg := &config.Config{}
	cfg.LogsStorage = "none"
	cfg.StorageBucket = "testkube-artifacts"
	cfg.LogsBucket = "testkube-logs"

	buckets := buildBuckets(cfg)
	assert.Len(t, buckets, 1)
	assert.Equal(t, "testkube-artifacts", buckets[0].name)
	assert.Equal(t, "storage", buckets[0].label)
}

// TestBuildBuckets_BothEnabled confirms that both buckets are included when
// neither storage option is disabled.
func TestBuildBuckets_BothEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.StorageBucket = "testkube-artifacts"
	cfg.LogsBucket = "testkube-logs"

	buckets := buildBuckets(cfg)
	assert.Len(t, buckets, 2)
	assert.Equal(t, "testkube-artifacts", buckets[0].name)
	assert.Equal(t, "storage", buckets[0].label)
	assert.Equal(t, "testkube-logs", buckets[1].name)
	assert.Equal(t, "logs", buckets[1].label)
}

// TestEnsureBucketsWithRetry_NoBuckets confirms that when both ARTIFACTS_STORAGE=none and
// LOGS_STORAGE=none produce an empty bucket list, ensureBucketsWithRetry makes no
// storage calls whatsoever.
func TestEnsureBucketsWithRetry_NoBuckets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// No BucketExists or CreateBucket calls are expected.
	mockClient := domainstorage.NewMockClient(ctrl)

	cfg := &config.Config{}
	cfg.ArtifactsStorage = "none"
	cfg.LogsStorage = "none"
	cfg.StorageBucket = "testkube-artifacts"
	cfg.LogsBucket = "testkube-logs"

	buckets := buildBuckets(cfg)
	assert.Empty(t, buckets)

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, buckets)
}

// TestEnsureBucketsWithRetry_BucketAlreadyExists verifies that when a bucket
// already exists, CreateBucket is not called.
func TestEnsureBucketsWithRetry_BucketAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "my-bucket").Return(true, nil)

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, []bucketSpec{{name: "my-bucket", label: "storage"}})
}

// TestEnsureBucketsWithRetry_CreatesBucket verifies that CreateBucket is called
// when the bucket does not yet exist.
func TestEnsureBucketsWithRetry_CreatesBucket(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "my-bucket").Return(false, nil)
	mockClient.EXPECT().CreateBucket(gomock.Any(), "my-bucket").Return(nil)

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, []bucketSpec{{name: "my-bucket", label: "storage"}})
}

// TestEnsureBucketsWithRetry_ContextCancelled verifies that the retry loop
// stops when the context is cancelled.
func TestEnsureBucketsWithRetry_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "my-bucket").
		DoAndReturn(func(_ context.Context, _ string) (bool, error) {
			cancel() // cancel the context on first attempt
			return false, errors.New("connection refused")
		})

	ensureBucketsWithRetry(ctx, mockClient, []bucketSpec{{name: "my-bucket", label: "storage"}})
}
