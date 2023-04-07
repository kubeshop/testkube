package minio

import (
	"context"
	"fmt"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
	"io"
	"strings"
)

type ArtifactClient struct {
	Endpoint        string
	accessKeyID     string
	secretAccessKey string
	ssl             bool
	region          string
	token           string
	bucket          string
	minioclient     *minio.Client
	Log             *zap.SugaredLogger
}

// NewMinIOArtifactClient returns new MinIO client
func NewMinIOArtifactClient(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) *ArtifactClient {
	c := &ArtifactClient{
		region:          region,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		token:           token,
		ssl:             ssl,
		bucket:          bucket,
		Endpoint:        endpoint,
		Log:             log.DefaultLogger,
	}

	return c
}

// ListFiles lists available files in the bucket from the config
func (c *ArtifactClient) ListFiles(ctx context.Context, executionId, testName, testSuiteName string) ([]testkube.Artifact, error) {
	c.Log.Infow("listing files", "bucket", c.bucket, "bucketFolder", executionId)
	// TODO: this is for back compatibility, remove it sometime in the future
	if exist, err := c.minioclient.BucketExists(ctx, executionId); err == nil && exist {
		formerResult, err := c.listFiles(ctx, executionId, "")
		if err == nil && len(formerResult) > 0 {
			return formerResult, nil
		}
	}

	return c.listFiles(ctx, c.bucket, executionId)
}

// listFiles lists available files in given bucket
func (c *ArtifactClient) listFiles(ctx context.Context, bucket, bucketFolder string) ([]testkube.Artifact, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	var toReturn []testkube.Artifact

	exists, err := c.minioclient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		c.Log.Debugw("bucket doesn't exist", "bucket", bucket)
		return nil, ErrArtifactsNotFound
	}
	listOptions := minio.ListObjectsOptions{Recursive: true}
	if bucketFolder != "" {
		listOptions.Prefix = bucketFolder
	}

	for obj := range c.minioclient.ListObjects(ctx, bucket, listOptions) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		if bucketFolder != "" {
			obj.Key = strings.TrimPrefix(obj.Key, bucketFolder+"/")
		}
		toReturn = append(toReturn, testkube.Artifact{Name: obj.Key, Size: int32(obj.Size)})
	}

	return toReturn, nil
}

// DownloadFile downloads file from bucket from the config
func (c *ArtifactClient) DownloadFile(ctx context.Context, file, executionId, testName, testSuiteName string) (io.Reader, error) {
	c.Log.Infow("Download file", "bucket", c.bucket, "bucketFolder", executionId, "file", file)
	// TODO: this is for back compatibility, remove it sometime in the future
	var errFirst error
	exists, err := c.minioclient.BucketExists(ctx, executionId)
	c.Log.Debugw("Checking if bucket exists", exists, err)
	if err == nil && exists {
		c.Log.Infow("Bucket exists, trying to get files from former bucket per execution", exists, err)
		objFirst, errFirst := c.downloadFile(ctx, executionId, "", file)
		if errFirst == nil && objFirst != nil {
			return objFirst, nil
		}
	}
	objSecond, errSecond := c.downloadFile(ctx, c.bucket, executionId, file)
	if errSecond != nil {
		return nil, fmt.Errorf("minio DownloadFile error: %v, error from getting files from former bucket per execution: %v", errSecond, errFirst)
	}
	return objSecond, nil
}

// downloadFile downloads file from bucket
func (c *ArtifactClient) downloadFile(ctx context.Context, bucket, bucketFolder, file string) (*minio.Object, error) {
	c.Log.Debugw("downloadFile", "bucket", bucket, "bucketFolder", bucketFolder, "file", file)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadFile .Connect error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		c.Log.Infow("bucket doesn't exist", "bucket", bucket)
		return nil, ErrArtifactsNotFound
	}

	if bucketFolder != "" {
		file = strings.Trim(bucketFolder, "/") + "/" + file
	}

	reader, err := c.minioclient.GetObject(ctx, bucket, file, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio DownloadFile GetObject error: %w", err)
	}

	_, err = reader.Stat()
	if err != nil {
		return reader, fmt.Errorf("minio Download File Stat error: %w", err)
	}

	return reader, nil
}

// Connect connects to MinIO server
func (c *ArtifactClient) Connect() error {
	creds := credentials.NewIAM("")
	c.Log.Debugw("connecting to minio",
		"endpoint", c.Endpoint,
		"accessKeyID", c.accessKeyID,
		"region", c.region,
		"token", c.token,
		"ssl", c.ssl)
	if c.accessKeyID != "" && c.secretAccessKey != "" {
		creds = credentials.NewStaticV4(c.accessKeyID, c.secretAccessKey, c.token)
	}
	opts := &minio.Options{
		Creds:  creds,
		Secure: c.ssl,
	}
	if c.region != "" {
		opts.Region = c.region
	}
	mclient, err := minio.New(c.Endpoint, opts)
	if err != nil {
		c.Log.Errorw("error connecting to minio", "error", err)
		return err
	}
	c.minioclient = mclient
	return err
}

var _ storage.ArtifactsStorage = (*ArtifactClient)(nil)
