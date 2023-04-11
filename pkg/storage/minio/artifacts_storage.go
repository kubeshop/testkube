package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/archive"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
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

	return c.downloadFile(ctx, c.bucket, executionId, file)
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

// DownloadArrchive downloads archive from bucket from the config
func (c *ArtifactClient) DownloadArchive(ctx context.Context, executionId string, masks []string) (io.Reader, error) {
	c.Log.Debugw("Downloading archive", "bucket", c.bucket, "bucketFolder", executionId, "masks", masks)

	return c.downloadArchive(ctx, c.bucket, executionId, masks)
}

// downloadArchive downloads archive from bucket
func (c *ArtifactClient) downloadArchive(ctx context.Context, bucket, bucketFolder string, masks []string) (io.Reader, error) {
	c.Log.Debugw("downloadArchive", "bucket", bucket, "bucketFolder", bucketFolder, "masks", masks)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadArchive .Connect error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		c.Log.Infow("bucket doesn't exist", "bucket", bucket)
		return nil, ErrArtifactsNotFound
	}

	var regexps []*regexp.Regexp
	for _, mask := range masks {
		re, err := regexp.Compile(mask)
		if err != nil {
			return nil, fmt.Errorf("minio DownloadArchive regexp error: %w", err)
		}

		regexps = append(regexps, re)
	}

	listOptions := minio.ListObjectsOptions{Recursive: true}
	if bucketFolder != "" {
		listOptions.Prefix = strings.Trim(bucketFolder, "/")
	}

	var files []*archive.File
	for obj := range c.minioclient.ListObjects(ctx, bucket, listOptions) {
		if obj.Err != nil {
			return nil, fmt.Errorf("minio DownloadArchive ListObjects error: %w", obj.Err)
		}

		found := len(regexps) == 0
		for i := range regexps {
			if found = regexps[i].MatchString(obj.Key); found {
				break
			}
		}

		if !found {
			continue
		}

		files = append(files, &archive.File{
			Name:    obj.Key,
			Size:    obj.Size,
			Mode:    int64(os.ModePerm),
			ModTime: obj.LastModified,
		})
	}

	for i := range files {
		reader, err := c.minioclient.GetObject(ctx, bucket, files[i].Name, minio.GetObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("minio DownloadArchive GetObject error: %w", err)
		}

		if _, err = reader.Stat(); err != nil {
			return nil, fmt.Errorf("minio DownloadArchive Stat error: %w", err)
		}

		files[i].Data = &bytes.Buffer{}
		if _, err = files[i].Data.ReadFrom(reader); err != nil {
			return nil, fmt.Errorf("minio DownloadArchive Read error: %w", err)
		}
	}

	service := archive.NewTarballService()
	data := &bytes.Buffer{}
	if err = service.Create(data, files); err != nil {
		return nil, fmt.Errorf("minio DownloadArchive CreateArchive error: %w", err)
	}

	return data, nil
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
