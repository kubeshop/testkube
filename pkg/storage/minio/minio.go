package minio

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var _ storage.Client = (*Client)(nil)

// ErrArtifactsNotFound contains error for not existing artifacts
var ErrArtifactsNotFound = errors.New("Execution doesn't have any artifacts associated with it")

// Client for managing MinIO storage server
type Client struct {
	Endpoint        string
	accessKeyID     string
	secretAccessKey string
	ssl             bool
	location        string
	token           string
	minioclient     *minio.Client
	Log             *zap.SugaredLogger
}

// NewClient returns new MinIO client
func NewClient(endpoint, accessKeyID, secretAccessKey, location, token string, ssl bool) *Client {
	c := &Client{
		location:        location,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		token:           token,
		ssl:             ssl,
		Endpoint:        endpoint,
		Log:             log.DefaultLogger,
	}

	return c
}

// Connect connects to MinIO server
func (c *Client) Connect() error {
	creds := credentials.NewIAM("")
	c.Log.Infow("connecting to minio", "endpoint", c.Endpoint, "accessKeyID", c.accessKeyID, "location", c.location, "token", c.token, "ssl", c.ssl)
	if c.accessKeyID != "" && c.secretAccessKey != "" {
		creds = credentials.NewStaticV4(c.accessKeyID, c.secretAccessKey, c.token)
	}
	mclient, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  creds,
		Secure: c.ssl,
	})
	if err != nil {
		c.Log.Errorw("error connecting to minio", "error", err)
		return err
	}
	c.minioclient = mclient
	return err
}

// CreateBucket creates new S3 like bucket
func (c *Client) CreateBucket(bucket string) error {
	if err := c.Connect(); err != nil {
		return err
	}
	ctx := context.Background()
	err := c.minioclient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: c.location})
	if err != nil {
		c.Log.Errorw("error creating bucket", "error", err)
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := c.minioclient.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			return fmt.Errorf("bucket %q already exists", bucket)
		} else {
			return err
		}
	}
	return nil
}

// DeleteBucket deletes bucket by name
func (c *Client) DeleteBucket(bucket string, force bool) error {
	if err := c.Connect(); err != nil {
		return err
	}
	return c.minioclient.RemoveBucketWithOptions(context.TODO(), bucket, minio.RemoveBucketOptions{ForceDelete: force})
}

// ListBuckets lists available buckets
func (c *Client) ListBuckets() ([]string, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	toReturn := []string{}
	if buckets, err := c.minioclient.ListBuckets(context.TODO()); err != nil {
		return nil, err
	} else {
		for _, bucket := range buckets {
			toReturn = append(toReturn, bucket.Name)
		}
	}
	return toReturn, nil
}

// ListFiles lists available files in given bucket
func (c *Client) ListFiles(bucket string) ([]testkube.Artifact, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	toReturn := []testkube.Artifact{}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrArtifactsNotFound
	}

	for obj := range c.minioclient.ListObjects(context.TODO(), bucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		toReturn = append(toReturn, testkube.Artifact{Name: obj.Key, Size: int32(obj.Size)})
	}

	return toReturn, nil
}

// SaveFile saves file defined by local filePath to S3 bucket
func (c *Client) SaveFile(bucket, filePath string) error {
	if err := c.Connect(); err != nil {
		return err
	}
	object, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("minio saving file (%s) open error: %w", filePath, err)
	}
	defer object.Close()
	objectStat, err := object.Stat()
	if err != nil {
		return fmt.Errorf("minio object stat (file:%s) error: %w", filePath, err)
	}

	fileName := objectStat.Name()

	c.Log.Debugw("saving object in minio", "filePath", filePath, "fileName", fileName, "bucket", bucket, "size", objectStat.Size())
	_, err = c.minioclient.PutObject(context.Background(), bucket, fileName, object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", fileName, err)
	}

	return nil
}

// DownloadFile downloads file in bucket
func (c *Client) DownloadFile(bucket, file string) (*minio.Object, error) {
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadFile .Connect error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrArtifactsNotFound
	}

	reader, err := c.minioclient.GetObject(context.Background(), bucket, file, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio DownloadFile GetObject error: %w", err)
	}

	_, err = reader.Stat()
	if err != nil {
		return reader, fmt.Errorf("minio Download File Stat error: %w", err)
	}

	return reader, nil
}

// ScrapeArtefacts pushes local files located in directories to given bucket ID
func (c *Client) ScrapeArtefacts(id string, directories ...string) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio scrape artefacts connection error: %w", err)
	}

	err := c.CreateBucket(id) // create bucket name it by execution ID
	if err != nil {
		return fmt.Errorf("minio failed to create a bucket %s: %w", id, err)
	}

	for _, directory := range directories {

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			c.Log.Debugw("directory %s does not exist, skipping", directory)
			continue
		}

		// if directory exists walk through recursively
		err = filepath.Walk(directory,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("minio path (%s) walk error: %w", path, err)
				}

				if !info.IsDir() {
					err = c.SaveFile(id, path) //The function will detect if there is a subdirectory and store accordingly
					if err != nil {
						return fmt.Errorf("minio save file (%s) error: %w", path, err)
					}
				}
				return nil
			})

		if err != nil {
			return fmt.Errorf("minio walk error: %w", err)
		}
	}
	return nil
}

// UploadFile saves a file to be copied into a running execution
func (c *Client) UploadFile(bucket string, filePath string, reader io.Reader, objectSize int64) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio UploadFile connection error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
	if err != nil {
		return fmt.Errorf("could not check if bucket already exists for copy files: %w", err)
	}

	if !exists {
		c.Log.Debugw("creating minio bucket for copy files", "bucket", bucket)
		err := c.CreateBucket(bucket)
		if err != nil {
			return fmt.Errorf("could not create bucket: %w", err)
		}
	}

	c.Log.Debugw("saving object in minio", "file", filePath, "bucket", bucket)
	_, err = c.minioclient.PutObject(context.Background(), bucket, filePath, reader, objectSize, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", filePath, err)
	}

	return nil
}

// PlaceFiles saves the content of the buckets to the filesystem
func (c *Client) PlaceFiles(buckets []string, prefix string) error {
	output.PrintLog(fmt.Sprintf("%s Getting the contents of buckets %s", ui.IconFile, buckets))
	if err := c.Connect(); err != nil {
		output.PrintLog(fmt.Sprintf("%s Minio PlaceFiles connection error: %s", ui.IconWarning, err.Error()))
		return fmt.Errorf("minio PlaceFiles connection error: %w", err)
	}

	for _, b := range buckets {
		exists, err := c.minioclient.BucketExists(context.TODO(), b)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Could not check if bucket already exists for files %s", ui.IconWarning, err.Error()))
			return fmt.Errorf("could not check if bucket already exists for files: %w", err)
		}
		if !exists {
			output.PrintLog(fmt.Sprintf("%s Bucket %s does not exist", ui.IconFile, b))
			continue
		}

		files, err := c.ListFiles(b)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Could not list files in bucket %s", ui.IconWarning, b))
			return fmt.Errorf("could not list files in bucket %s", b)
		}

		for _, f := range files {
			output.PrintEvent(fmt.Sprintf("%s Downloading file %s", ui.IconFile, f.Name))
			c.Log.Infof("Getting file %s", f)
			err = c.minioclient.FGetObject(context.Background(), b, f.Name, prefix+f.Name, minio.GetObjectOptions{})
			if err != nil {
				output.PrintEvent(fmt.Sprintf("%s Could not download file %s", ui.IconCross, f.Name))
				return fmt.Errorf("could not persist file %s from bucket %s: %w", f.Name, b, err)
			}
			output.PrintEvent(fmt.Sprintf("%s File %s successfully downloaded into %s", ui.IconCheckMark, f.Name, prefix))
		}
	}
	return nil
}

// GetValidBucketName returns a minio-compatible bucket name
func (c *Client) GetValidBucketName(parentType string, parentName string) string {
	bucketName := fmt.Sprintf("%s-%s", parentType, parentName)
	if len(bucketName) <= 63 {
		return bucketName
	}

	h := fnv.New32a()
	h.Write([]byte(bucketName))

	return fmt.Sprintf("%s-%d", bucketName[:52], h.Sum32())
}

func (c *Client) DeleteFile(bucket, file string) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio DeleteFile connection error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
	if err != nil {
		return fmt.Errorf("could not check if bucket already exists for delete file: %w", err)
	}

	if !exists {
		return ErrArtifactsNotFound
	}

	err = c.minioclient.RemoveObject(context.Background(), bucket, file, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("minio DeleteFile RemoveObject error: %w", err)
	}

	return nil
}
