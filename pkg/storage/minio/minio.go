package minio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var _ storage.Client = (*Client)(nil)

type Client struct {
	Endpoint        string
	accessKeyID     string
	secretAccessKey string
	ssl             bool
	location        string
	token           string
	minioclient     *minio.Client
}

func NewClient(endpoint, accessKeyID, secretAccessKey, location, token string, ssl bool) (*Client, error) {
	c := &Client{
		location:        location,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		token:           token,
		ssl:             ssl,
	}

	mclient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, token),
		Secure: ssl,
	})
	if err != nil {
		return nil, err
	}
	c.minioclient = mclient

	return c, nil
}

func (c *Client) CreateBucket(bucket string) error {
	ctx := context.Background()
	err := c.minioclient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: c.location})
	if err != nil {
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

func (c *Client) DeleteBucket(bucket string, force bool) error {
	return c.minioclient.RemoveBucketWithOptions(context.TODO(), bucket, minio.BucketOptions{ForceDelete: force})
}

func (c *Client) ListBuckets() ([]string, error) {
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

func (c *Client) ListFiles(bucket string) ([]testkube.Artifact, error) {
	toReturn := []testkube.Artifact{}
	for obj := range c.minioclient.ListObjects(context.TODO(), bucket, minio.ListObjectsOptions{}) {
		if obj.Err != nil {
			return nil, obj.Err
		}

		toReturn = append(toReturn, testkube.Artifact{Name: obj.Key, Size: int32(obj.Size)})
	}

	return toReturn, nil
}

func (c *Client) SaveFile(bucket, filePath string) error {
	object, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer object.Close()
	objectStat, err := object.Stat()
	if err != nil {
		return err
	}

	fileName := objectStat.Name()

	n, err := c.minioclient.PutObject(context.Background(), bucket, fileName, object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		return err
	}

	fmt.Printf("uploaded %q of size %d\n", filePath, n.Size)
	return nil
}

func (c *Client) DownloadFile(bucket, file string) (*minio.Object, error) {
	reader, err := c.minioclient.GetObject(context.Background(), bucket, file, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (c *Client) ScrapeArtefacts(id, directory string) error {
	client := c
	err := client.CreateBucket(id) // create bucket name it by execution ID
	if err != nil {
		return fmt.Errorf("failed to create a bucket %s: %w", id, err)
	}
	err = filepath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				// if strings.Contains(path, "/") {
				// 	pth := strings.Split(path, "/")
				// 	path = pth[0]
				// }
				err = client.SaveFile(id, path) //The function will detect if there is a subdirectory and store accordingly
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return nil
}
