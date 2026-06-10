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

func TestEnsureBucketsWithRetry_SkipsEmptyBucketNames(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)

	// No calls should be made when all bucket names are empty
	buckets := []bucketSpec{
		{name: "", label: "storage"},
		{name: "", label: "logs"},
	}

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, buckets)
}

func TestEnsureBucketsWithRetry_CreatesBucketIfNotExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)

	mockClient.EXPECT().BucketExists(gomock.Any(), "testkube-artifacts").Return(false, nil)
	mockClient.EXPECT().CreateBucket(gomock.Any(), "testkube-artifacts").Return(nil)

	buckets := []bucketSpec{
		{name: "testkube-artifacts", label: "storage"},
	}

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, buckets)
}

func TestEnsureBucketsWithRetry_SkipsBucketIfAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)

	mockClient.EXPECT().BucketExists(gomock.Any(), "testkube-artifacts").Return(true, nil)

	buckets := []bucketSpec{
		{name: "testkube-artifacts", label: "storage"},
	}

	ctx := context.Background()
	ensureBucketsWithRetry(ctx, mockClient, buckets)
}

func TestEnsureBucketsWithRetry_CancelledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)

	// First attempt fails
	mockClient.EXPECT().BucketExists(gomock.Any(), "testkube-artifacts").Return(false, errors.New("connection refused"))

	buckets := []bucketSpec{
		{name: "testkube-artifacts", label: "storage"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately so it exits after first failed attempt
	ensureBucketsWithRetry(ctx, mockClient, buckets)
}

func TestBucketListConstruction_LogsStorageNone(t *testing.T) {
	cfg := &config.Config{
		OSSControlPlaneConfig: config.OSSControlPlaneConfig{
			LogsStorage:   "none",
			StorageBucket: "testkube-artifacts",
			LogsBucket:    "testkube-logs",
		},
	}

	// Simulate the bucket list construction logic from CreateControlPlane
	buckets := []bucketSpec{{name: cfg.StorageBucket, label: "storage"}}
	if cfg.LogsStorage != "none" {
		buckets = append(buckets, bucketSpec{name: cfg.LogsBucket, label: "logs"})
	}

	assert.Len(t, buckets, 1)
	assert.Equal(t, "testkube-artifacts", buckets[0].name)
	assert.Equal(t, "storage", buckets[0].label)
}

func TestBucketListConstruction_LogsStorageMinio(t *testing.T) {
	cfg := &config.Config{
		OSSControlPlaneConfig: config.OSSControlPlaneConfig{
			LogsStorage:   "minio",
			StorageBucket: "testkube-artifacts",
			LogsBucket:    "testkube-logs",
		},
	}

	// Simulate the bucket list construction logic from CreateControlPlane
	buckets := []bucketSpec{{name: cfg.StorageBucket, label: "storage"}}
	if cfg.LogsStorage != "none" {
		buckets = append(buckets, bucketSpec{name: cfg.LogsBucket, label: "logs"})
	}

	assert.Len(t, buckets, 2)
	assert.Equal(t, "testkube-artifacts", buckets[0].name)
	assert.Equal(t, "testkube-logs", buckets[1].name)
}

func TestBucketListConstruction_LogsStorageEmpty(t *testing.T) {
	cfg := &config.Config{
		OSSControlPlaneConfig: config.OSSControlPlaneConfig{
			LogsStorage:   "",
			StorageBucket: "testkube-artifacts",
			LogsBucket:    "testkube-logs",
		},
	}

	// When LogsStorage is empty (default), logs bucket should still be ensured
	buckets := []bucketSpec{{name: cfg.StorageBucket, label: "storage"}}
	if cfg.LogsStorage != "none" {
		buckets = append(buckets, bucketSpec{name: cfg.LogsBucket, label: "logs"})
	}

	assert.Len(t, buckets, 2)
	assert.Equal(t, "testkube-artifacts", buckets[0].name)
	assert.Equal(t, "storage", buckets[0].label)
	assert.Equal(t, "testkube-logs", buckets[1].name)
	assert.Equal(t, "logs", buckets[1].label)
}

func TestEnsureBucket_BucketExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "test-bucket").Return(true, nil)

	result := ensureBucket(context.Background(), mockClient, bucketSpec{name: "test-bucket", label: "test"})
	assert.True(t, result)
}

func TestEnsureBucket_BucketCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "test-bucket").Return(false, nil)
	mockClient.EXPECT().CreateBucket(gomock.Any(), "test-bucket").Return(nil)

	result := ensureBucket(context.Background(), mockClient, bucketSpec{name: "test-bucket", label: "test"})
	assert.True(t, result)
}

func TestEnsureBucket_BucketExistsCheckFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "test-bucket").Return(false, errors.New("connection error"))

	result := ensureBucket(context.Background(), mockClient, bucketSpec{name: "test-bucket", label: "test"})
	assert.False(t, result)
}

func TestEnsureBucket_CreateBucketFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "test-bucket").Return(false, nil)
	mockClient.EXPECT().CreateBucket(gomock.Any(), "test-bucket").Return(errors.New("access denied"))

	result := ensureBucket(context.Background(), mockClient, bucketSpec{name: "test-bucket", label: "test"})
	assert.False(t, result)
}

func TestEnsureBucket_AlreadyExistsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := domainstorage.NewMockClient(ctrl)
	mockClient.EXPECT().BucketExists(gomock.Any(), "test-bucket").Return(false, nil)
	mockClient.EXPECT().CreateBucket(gomock.Any(), "test-bucket").Return(errors.New("bucket already exists"))

	result := ensureBucket(context.Background(), mockClient, bucketSpec{name: "test-bucket", label: "test"})
	assert.True(t, result)
}
