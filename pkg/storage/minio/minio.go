package minio

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/archive"
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
	region         string
	bucket         string
	minioClient    *minio.Client
	Log            *zap.SugaredLogger
	minioConnecter *Connecter
}

// NewClient returns new MinIO client
func NewClient(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, opts ...Option) *Client {
	c := &Client{
		minioConnecter: NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket, log.DefaultLogger, opts...),
		region:         region,
		bucket:         bucket,
		Log:            log.DefaultLogger,
	}

	return c
}

// Connect connects to MinIO server
func (c *Client) Connect() error {
	var err error
	c.minioClient, err = c.minioConnecter.GetClient()
	return err
}

func (c *Client) SetExpirationPolicy(expirationDays int) error {
	if expirationDays != 0 && c.minioClient != nil {
		lifecycleConfig := &lifecycle.Configuration{
			Rules: []lifecycle.Rule{
				{
					ID:     "expiration_policy",
					Status: "Enabled",
					Expiration: lifecycle.Expiration{
						Days: lifecycle.ExpirationDays(expirationDays),
					},
				},
			},
		}
		return c.minioClient.SetBucketLifecycle(context.TODO(), c.bucket, lifecycleConfig)
	}
	return nil
}

// CreateBucket creates new S3 like bucket
func (c *Client) CreateBucket(ctx context.Context, bucket string) error {
	if err := c.Connect(); err != nil {
		return err
	}
	err := c.minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: c.region})
	if err != nil {
		c.Log.Errorw("error creating bucket", "error", err)
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := c.minioClient.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			return fmt.Errorf("bucket %q already exists", bucket)
		} else {
			return err
		}
	}
	return nil
}

// DeleteBucket deletes bucket by name
func (c *Client) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	if err := c.Connect(); err != nil {
		return err
	}
	return c.minioClient.RemoveBucketWithOptions(ctx, bucket, minio.RemoveBucketOptions{ForceDelete: force})
}

// ListBuckets lists available buckets
func (c *Client) ListBuckets(ctx context.Context) ([]string, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	var toReturn []string
	if buckets, err := c.minioClient.ListBuckets(ctx); err != nil {
		return nil, err
	} else {
		for _, bucket := range buckets {
			toReturn = append(toReturn, bucket.Name)
		}
	}
	return toReturn, nil
}

// listFiles lists available files in given bucket
func (c *Client) listFiles(ctx context.Context, bucket, bucketFolder string) ([]testkube.Artifact, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	var toReturn []testkube.Artifact

	exists, err := c.minioClient.BucketExists(ctx, bucket)
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

	for obj := range c.minioClient.ListObjects(ctx, bucket, listOptions) {
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
func (c *Client) ListFiles(ctx context.Context, bucketFolder string) ([]testkube.Artifact, error) {
	c.Log.Infow("listing files", "bucket", c.bucket, "bucketFolder", bucketFolder)
	// TODO: this is for back compatibility, remove it sometime in the future
	if bucketFolder != "" {
		if exist, err := c.minioClient.BucketExists(ctx, bucketFolder); err == nil && exist {
			formerResult, err := c.listFiles(ctx, bucketFolder, "")
			if err == nil && len(formerResult) > 0 {
				return formerResult, nil
			}
		}
	}

	return c.listFiles(ctx, c.bucket, bucketFolder)
}

// saveFile saves file defined by local filePath to S3 bucket
func (c *Client) saveFile(ctx context.Context, bucket, bucketFolder, filePath string) error {
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

	exists, err := c.minioClient.BucketExists(ctx, bucket)
	if err != nil || !exists {
		err := c.CreateBucket(ctx, bucket)
		if err != nil {
			return fmt.Errorf("minio saving file (%s) bucket was not created and create bucket returnes error: %w", filePath, err)
		}
	}

	fileName := objectStat.Name()
	if bucketFolder != "" {
		fileName = strings.Trim(bucketFolder, "/") + "/" + fileName
	}

	c.Log.Debugw("saving object in minio", "filePath", filePath, "fileName", fileName, "bucket", bucket, "size", objectStat.Size())
	_, err = c.minioClient.PutObject(ctx, bucket, fileName, object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", fileName, err)
	}

	return nil
}

func (c *Client) SaveFileDirect(ctx context.Context, folder, file string, data io.Reader, size int64, opts minio.PutObjectOptions) error {
	exists, err := c.minioClient.BucketExists(ctx, c.bucket)
	if err != nil {
		return errors.Wrapf(err, "error checking does bucket %s exists", c.bucket)
	}
	if !exists {
		if err := c.CreateBucket(ctx, c.bucket); err != nil {
			return errors.Wrapf(err, "error creating bucket %s", c.bucket)
		}
	}

	filename := file
	if folder != "" {
		filename = fmt.Sprintf("%s/%s", folder, filename)
	}

	if opts.ContentType == "" {
		opts.ContentType = "application/octet-stream"
	}
	c.Log.Debugw("saving object in minio", "filename", filename, "bucket", c.bucket, "size", size)
	_, err = c.minioClient.PutObject(ctx, c.bucket, filename, data, size, opts)
	if err != nil {
		return errors.Wrapf(err, "minio saving file (%s) put object error", filename)
	}

	return nil
}

// SaveFile saves file defined by local filePath to S3 bucket from the config
func (c *Client) SaveFile(ctx context.Context, bucketFolder, filePath string) error {
	c.Log.Debugw("SaveFile", "bucket", c.bucket, "bucketFolder", bucketFolder, "filePath", filePath)
	return c.saveFile(ctx, c.bucket, bucketFolder, filePath)
}

// downloadFile downloads file from bucket
func (c *Client) downloadFile(ctx context.Context, bucket, bucketFolder, file string) (*minio.Object, error) {
	c.Log.Debugw("downloadFile", "bucket", bucket, "bucketFolder", bucketFolder, "file", file)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadFile .Connect error: %w", err)
	}

	exists, err := c.minioClient.BucketExists(ctx, bucket)
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

	reader, err := c.minioClient.GetObject(ctx, bucket, file, minio.GetObjectOptions{})
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
func (c *Client) DownloadFile(ctx context.Context, bucketFolder, file string) (*minio.Object, error) {
	c.Log.Infow("Download file", "bucket", c.bucket, "bucketFolder", bucketFolder, "file", file)
	// TODO: this is for back compatibility, remove it sometime in the future
	var objFirst *minio.Object
	var errFirst error
	if bucketFolder != "" {
		exists, err := c.minioClient.BucketExists(ctx, bucketFolder)
		c.Log.Debugw("Checking if bucket exists", exists, err)
		if err == nil && exists {
			c.Log.Infow("Bucket exists, trying to get files from former bucket per execution", exists, err)
			objFirst, errFirst = c.downloadFile(ctx, bucketFolder, "", file)
			if errFirst == nil && objFirst != nil {
				return objFirst, nil
			}
		}
	}
	objSecond, errSecond := c.downloadFile(ctx, c.bucket, bucketFolder, file)
	if errSecond != nil {
		return nil, fmt.Errorf("minio DownloadFile error: %v, error from getting files from former bucket per execution: %v", errSecond, errFirst)
	}
	return objSecond, nil
}

// downloadArchive downloads archive from bucket
func (c *Client) downloadArchive(ctx context.Context, bucket, bucketFolder string, masks []string) (io.Reader, error) {
	c.Log.Debugw("downloadArchive", "bucket", bucket, "bucketFolder", bucketFolder, "masks", masks)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("minio DownloadArchive .Connect error: %w", err)
	}

	exists, err := c.minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		c.Log.Infow("bucket doesn't exist", "bucket", bucket)
		return nil, ErrArtifactsNotFound
	}

	var regexps []*regexp.Regexp
	for _, mask := range masks {
		values := strings.Split(mask, ",")
		for _, value := range values {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("minio DownloadArchive regexp error: %w", err)
			}

			regexps = append(regexps, re)
		}
	}

	listOptions := minio.ListObjectsOptions{Recursive: true}
	if bucketFolder != "" {
		listOptions.Prefix = strings.Trim(bucketFolder, "/")
	}

	var files []*archive.File
	for obj := range c.minioClient.ListObjects(ctx, bucket, listOptions) {
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
		reader, err := c.minioClient.GetObject(ctx, bucket, files[i].Name, minio.GetObjectOptions{})
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

// DownloadArchive downloads archive from bucket from the config
func (c *Client) DownloadArchive(ctx context.Context, bucketFolder string, masks []string) (io.Reader, error) {
	c.Log.Infow("Download archive", "bucket", c.bucket, "bucketFolder", bucketFolder, "masks", masks)
	// TODO: this is for back compatibility, remove it sometime in the future
	var objFirst io.Reader
	var errFirst error
	if bucketFolder != "" {
		exists, err := c.minioClient.BucketExists(ctx, bucketFolder)
		c.Log.Debugw("Checking if bucket exists", exists, err)
		if err == nil && exists {
			c.Log.Infow("Bucket exists, trying to get archive from former bucket per execution", exists, err)
			objFirst, errFirst = c.downloadArchive(ctx, bucketFolder, "", masks)
			if errFirst == nil && objFirst != nil {
				return objFirst, nil
			}
		}
	}
	objSecond, errSecond := c.downloadArchive(ctx, c.bucket, bucketFolder, masks)
	if errSecond != nil {
		return nil, fmt.Errorf("minio DownloadArchive error: %v, error from getting archive from former bucket per execution: %v", errSecond, errFirst)
	}
	return objSecond, nil
}

// DownloadFileFromBucket downloads file from given bucket
func (c *Client) DownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) (io.Reader, minio.ObjectInfo, error) {
	c.Log.Debugw("Downloading file", "bucket", bucket, "bucketFolder", bucketFolder, "file", file)
	object, err := c.downloadFile(ctx, bucket, bucketFolder, file)
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}

	info, err := object.Stat()
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}

	return object, info, nil
}

// DownloadArrchiveFromBucket downloads archive from given bucket
func (c *Client) DownloadArchiveFromBucket(ctx context.Context, bucket, bucketFolder string, masks []string) (io.Reader, error) {
	c.Log.Debugw("Downloading archive", "bucket", bucket, "bucketFolder", bucketFolder, "masks", masks)
	return c.downloadArchive(ctx, bucket, bucketFolder, masks)
}

// ScrapeArtefacts pushes local files located in directories to given folder with given id located in the configured bucket
func (c *Client) ScrapeArtefacts(ctx context.Context, id string, directories ...string) error {
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
					err = c.SaveFile(ctx, id, path) //The function will detect if there is a subdirectory and store accordingly
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
func (c *Client) uploadFile(ctx context.Context, bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio UploadFile connection error: %w", err)
	}

	exists, err := c.minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("could not check if bucket already exists for copy files: %w", err)
	}

	if !exists {
		c.Log.Debugw("creating minio bucket for copy files", "bucket", bucket)
		err := c.CreateBucket(ctx, bucket)
		if err != nil {
			return fmt.Errorf("could not create bucket: %w", err)
		}
	}

	if bucketFolder != "" {
		filePath = strings.Trim(bucketFolder, "/") + "/" + filePath
	}

	c.Log.Debugw("saving object in minio", "file", filePath, "bucket", bucket)
	_, err = c.minioClient.PutObject(ctx, bucket, filePath, reader, objectSize, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("minio saving file (%s) put object error: %w", filePath, err)
	}

	return nil
}

// UploadFile saves a file to be copied into a running execution
func (c *Client) UploadFile(ctx context.Context, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return c.uploadFile(ctx, c.bucket, bucketFolder, filePath, reader, objectSize)
}

// UploadFileToBucket saves a file to be copied into a running execution
func (c *Client) UploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return c.uploadFile(ctx, bucket, bucketFolder, filePath, reader, objectSize)
}

// PlaceFiles saves the content of the buckets to the filesystem
func (c *Client) PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error {
	output.PrintLog(fmt.Sprintf("%s Getting the contents of bucket folders %s", ui.IconFile, bucketFolders))
	if err := c.Connect(); err != nil {
		output.PrintLog(fmt.Sprintf("%s Minio PlaceFiles connection error: %s", ui.IconWarning, err.Error()))
		return fmt.Errorf("minio PlaceFiles connection error: %w", err)
	}
	exists, err := c.minioClient.BucketExists(ctx, c.bucket)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Could not check if bucket already exists for files %s", ui.IconWarning, err.Error()))
		return fmt.Errorf("could not check if bucket already exists for files: %w", err)
	}
	if !exists {
		output.PrintLog(fmt.Sprintf("%s Bucket %s does not exist", ui.IconFile, c.bucket))
		return fmt.Errorf("bucket %s does not exist", c.bucket)
	}
	for _, folder := range bucketFolders {

		files, err := c.ListFiles(ctx, folder)
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Could not list files in bucket %s folder %s", ui.IconWarning, c.bucket, folder))
			return fmt.Errorf("could not list files in bucket %s folder %s", c.bucket, folder)
		}

		for _, f := range files {
			output.PrintEvent(fmt.Sprintf("%s Downloading file %s", ui.IconFile, f.Name))
			c.Log.Infof("Getting file %s", f)
			objectName := f.Name

			isFileDownloadable := strings.TrimSpace(objectName) != "" && !strings.HasSuffix(objectName, "/")
			if !isFileDownloadable {
				output.PrintEvent(fmt.Sprintf("%s File %s cannot be downloaded", ui.IconCross, objectName))
				continue
			}

			if folder != "" {
				objectName = fmt.Sprintf("%s/%s", folder, objectName)
			}

			path := filepath.Join(prefix, f.Name)
			err = c.minioClient.FGetObject(ctx, c.bucket, objectName, path, minio.GetObjectOptions{})
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

func (c *Client) deleteFile(ctx context.Context, bucket, bucketFolder, file string) error {
	if err := c.Connect(); err != nil {
		return fmt.Errorf("minio DeleteFile connection error: %w", err)
	}

	exists, err := c.minioClient.BucketExists(ctx, bucket)
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

	err = c.minioClient.RemoveObject(ctx, bucket, file, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("minio DeleteFile RemoveObject error: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from a bucket folder where bucket is provided by config
func (c *Client) DeleteFile(ctx context.Context, bucketFolder, file string) error {
	// TODO: this is for back compatibility, remove it sometime in the future
	var errFirst error
	if bucketFolder != "" {
		if exist, err := c.minioClient.BucketExists(ctx, bucketFolder); err != nil || !exist {
			errFirst = c.DeleteFileFromBucket(ctx, bucketFolder, "", file)
			if err == nil {
				return nil
			}

		}
	}
	errSecond := c.deleteFile(ctx, c.bucket, bucketFolder, file)
	if errFirst != nil {
		return fmt.Errorf("deleting file error: %v, previous attemt to delete file from one bucket per execution error: %v", errSecond, errFirst)
	}
	return errSecond
}

// DeleteFileFromBucket deletes a file from a bucket folder
func (c *Client) DeleteFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) error {
	return c.deleteFile(ctx, bucket, bucketFolder, file)
}

// IsConnectionPossible checks if the connection to minio is possible
func (c *Client) IsConnectionPossible(ctx context.Context) (bool, error) {
	if err := c.Connect(); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) PresignDownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string, expires time.Duration) (string, error) {
	if err := c.Connect(); err != nil {
		return "", err
	}
	if bucketFolder != "" {
		file = strings.Trim(bucketFolder, "/") + "/" + file
	}
	c.Log.Debugw("presigning get object from minio", "file", file, "bucket", bucket)
	url, err := c.minioClient.PresignedPutObject(ctx, bucket, file, expires)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (c *Client) PresignUploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, expires time.Duration) (string, error) {
	if err := c.Connect(); err != nil {
		return "", err
	}
	if bucketFolder != "" {
		filePath = strings.Trim(bucketFolder, "/") + "/" + filePath
	}
	c.Log.Debugw("presigning put object in minio", "file", filePath, "bucket", bucket)
	url, err := c.minioClient.PresignedPutObject(ctx, bucket, filePath, expires)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
