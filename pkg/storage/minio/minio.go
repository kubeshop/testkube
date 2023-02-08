package minio

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/ui"
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
	bucket          string
	minioclient     *minio.Client
	Log             *zap.SugaredLogger
}

// NewClient returns new MinIO client
func NewClient(endpoint, accessKeyID, secretAccessKey, location, token, bucket string, ssl bool) *Client {
	c := &Client{
		location:        location,
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

// Connect connects to MinIO server
func (c *Client) Connect() error {
	creds := credentials.NewIAM("")
	c.Log.Debugw("connecting to minio", "endpoint", c.Endpoint, "accessKeyID", c.accessKeyID, "location", c.location, "token", c.token, "ssl", c.ssl)
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

// listFiles lists available files in given bucket
func (c *Client) listFiles(bucket, bucketFolder string) ([]testkube.Artifact, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	toReturn := []testkube.Artifact{}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
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

	for obj := range c.minioclient.ListObjects(context.TODO(), bucket, listOptions) {
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

// ListFiles lists available files in the bucket from the config
func (c *Client) ListFiles(bucketFolder string) ([]testkube.Artifact, error) {
	c.Log.Infow("listing files", "bucket", c.bucket, "bucketFolder", bucketFolder)
	// TODO: this is for back compatibility, remove it sometime in the future
	if exist, err := c.minioclient.BucketExists(context.TODO(), bucketFolder); err == nil && exist {
		formerResult, err := c.listFiles(bucketFolder, "")
		if err == nil && len(formerResult) > 0 {
			return formerResult, nil
		}
	}

	return c.listFiles(c.bucket, bucketFolder)
}

// ListFilesFromBucket lists available files in given bucket
func (c *Client) ListFilesFromBucket(bucket string) ([]testkube.Artifact, error) {
	return c.listFiles(bucket, "")
}

// saveFile saves file defined by local filePath to S3 bucket
func (c *Client) saveFile(bucket, bucketFolder, filePath string) error {
	c.Log.Debugw("saving file", "bucket", bucket, "bucketFolder", bucketFolder, "filePath", filePath)
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

	exists, err := c.minioclient.BucketExists(context.Background(), bucket)
	if err != nil || !exists {
		err := c.CreateBucket(bucket)
		if err != nil {
			return fmt.Errorf("minio saving file (%s) bucket was not created and create bucket returnes error: %w", filePath, err)
		}
	}

	fileName := strings.Trim(bucketFolder, "/") + "/" + objectStat.Name()

	c.Log.Debugw("saving object in minio", "filePath", filePath, "fileName", fileName, "bucket", bucket, "size", objectStat.Size())
	_, err = c.minioclient.PutObject(context.Background(), bucket, fileName, object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", fileName, err)
	}

	return nil
}

// SaveFile saves file defined by local filePath to S3 bucket from the config
func (c *Client) SaveFile(bucketFolder, filePath string) error {
	c.Log.Debugw("SaveFile", "bucket", c.bucket, "bucketFolder", bucketFolder, "filePath", filePath)
	return c.saveFile(c.bucket, bucketFolder, filePath)
}

// SaveFileToBucket saves file defined by local filePath to given S3 bucket
func (c *Client) SaveFileToBucket(bucket, bucketFolder, filePath string) error {
	c.Log.Debugw("SaveFileToBucket", "bucket", bucket, "bucketFolder", bucketFolder, "filePath", filePath)
	return c.saveFile(bucket, bucketFolder, filePath)
}

// downloadFile downloads file from bucket
func (c *Client) downloadFile(bucket, bucketFolder, file string) (*minio.Object, error) {
	c.Log.Infow("downloadFile", "bucket", bucket, "bucketFolder", bucketFolder, "file", file)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadFile .Connect error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
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

// DownloadFile downloads file from bucket from the config
func (c *Client) DownloadFile(bucketFolder, file string) (*minio.Object, error) {
	c.Log.Infow("Download file", "bucket", c.bucket, "bucketFolder", bucketFolder, "file", file)
	// TODO: this is for back compatibility, remove it sometime in the future
	var errFirst error
	exists, err := c.minioclient.BucketExists(context.TODO(), bucketFolder)
	c.Log.Debugw("Checking if bucket exists", exists, err)
	if err == nil && exists {
		c.Log.Infow("Bucket exists, trying to get files from former bucket per execution", exists, err)
		objFirst, errFirst := c.downloadFile(bucketFolder, "", file)
		if errFirst == nil && objFirst != nil {
			return objFirst, nil
		}
	}
	objSecond, errSecond := c.downloadFile(c.bucket, bucketFolder, file)
	if errSecond != nil {
		return nil, fmt.Errorf("minio DownloadFile error: %v, error from getting files from former bucket per execution: %v", errSecond, errFirst)
	}
	return objSecond, nil
}

// DownloadFileFromBucket downloads file from given bucket
func (c *Client) DownloadFileFromBucket(bucket, bucketFolder, file string) (*minio.Object, error) {
	c.Log.Infow("Downloading file", "bucket", bucket, "bucketFolder", bucketFolder, "file", file)
	return c.downloadFile(bucket, bucketFolder, file)
}

// ScrapeArtefacts pushes local files located in directories to given folder with given id located in the configured bucket
func (c *Client) ScrapeArtefacts(id string, directories ...string) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio scrape artefacts connection error: %w", err)
	}

	for _, directory := range directories {

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			c.Log.Debugw("directory %s does not exist, skipping", directory)
			continue
		}

		// if directory exists walk through recursively
		err := filepath.Walk(directory,
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

// uploadFile saves a file to be copied into a running execution
func (c *Client) uploadFile(bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
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

	if bucketFolder != "" {
		filePath = strings.Trim(bucketFolder, "/") + "/" + filePath
	}

	c.Log.Debugw("saving object in minio", "file", filePath, "bucket", bucket)
	_, err = c.minioclient.PutObject(context.Background(), bucket, filePath, reader, objectSize, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", filePath, err)
	}

	return nil
}

// UploadFile saves a file to be copied into a running execution
func (c *Client) UploadFile(bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return c.uploadFile(c.bucket, bucketFolder, filePath, reader, objectSize)
}

// UploadFileToBucket saves a file to be copied into a running execution
func (c *Client) UploadFileToBucket(bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return c.uploadFile(bucket, bucketFolder, filePath, reader, objectSize)
}

// laceFiles saves the content of the buckets to the filesystem
func (c *Client) PlaceFiles(bucketFolders []string, prefix string) error {
	output.PrintLog(fmt.Sprintf("%s Getting the contents of bucket folders %s", ui.IconFile, bucketFolders))
	if err := c.Connect(); err != nil {
		output.PrintLog(fmt.Sprintf("%s Minio PlaceFiles connection error: %s", ui.IconWarning, err.Error()))
		return fmt.Errorf("minio PlaceFiles connection error: %w", err)
	}
	exists, err := c.minioclient.BucketExists(context.TODO(), c.bucket)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Could not check if bucket already exists for files %s", ui.IconWarning, err.Error()))
		return fmt.Errorf("could not check if bucket already exists for files: %w", err)
	}
	if !exists {
		output.PrintLog(fmt.Sprintf("%s Bucket %s does not exist", ui.IconFile, c.bucket))
		return fmt.Errorf("bucket %s does not exist", c.bucket)
	}
	for _, folder := range bucketFolders {

		files, err := c.ListFiles(folder)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Could not list files in bucket %s folder %s", ui.IconWarning, c.bucket, folder))
			return fmt.Errorf("could not list files in bucket %s folder %s", c.bucket, folder)
		}

		for _, f := range files {
			output.PrintEvent(fmt.Sprintf("%s Downloading file %s", ui.IconFile, f.Name))
			c.Log.Infof("Getting file %s", f)
			err = c.minioclient.FGetObject(context.Background(), c.bucket, f.Name, prefix+f.Name, minio.GetObjectOptions{})
			if err != nil {
				output.PrintEvent(fmt.Sprintf("%s Could not download file %s", ui.IconCross, f.Name))
				return fmt.Errorf("could not persist file %s from bucket %s, folder %s: %w", f.Name, c.bucket, folder, err)
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

func (c *Client) deleteFile(bucket, bucketFolder, file string) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio DeleteFile connection error: %w", err)
	}

	exists, err := c.minioclient.BucketExists(context.TODO(), bucket)
	if err != nil {
		return fmt.Errorf("could not check if bucket already exists for delete file: %w", err)
	}

	if !exists {
		c.Log.Warnf("bucket %s does not exist", bucket)
		return ErrArtifactsNotFound
	}

	if bucketFolder != "" {
		file = strings.Trim(bucketFolder, "/") + "/" + file
	}

	err = c.minioclient.RemoveObject(context.Background(), bucket, file, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("minio DeleteFile RemoveObject error: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from a bucket folder where bucket is provided by config
func (c *Client) DeleteFile(bucketFolder, file string) error {
	// TODO: this is for back compatibility, remove it sometime in the future
	var errFirst error
	if exist, err := c.minioclient.BucketExists(context.TODO(), c.bucket); err != nil || !exist {
		errFirst = c.DeleteFileFromBucket(bucketFolder, "", file)
		if err == nil {
			return nil
		}

	}
	errSecond := c.deleteFile(c.bucket, bucketFolder, file)
	if errFirst != nil {
		return fmt.Errorf("deleting file error: %v, previous attemt to delete file from one bucket per execution error: %v", errSecond, errFirst)
	}
	return errSecond
}

// DeleteFileFromBucket deletes a file from a bucket folder
func (c *Client) DeleteFileFromBucket(bucket, bucketFolder, file string) error {
	return c.deleteFile(bucket, bucketFolder, file)
}
