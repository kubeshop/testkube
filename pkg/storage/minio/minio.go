package minio

import (
	"context"
	"fmt"
	"io"
	"os"

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

func (c *Client) ListFiles(bucket string) ([]string, error) {
	toReturn := []string{}
	for obj := range c.minioclient.ListObjects(context.TODO(), bucket, minio.ListObjectsOptions{}) {
		if obj.Err != nil {
			fmt.Println(obj.Err)
			return nil, obj.Err
		}
		toReturn = append(toReturn, obj.Key)
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

	var fileName string
	if strings.Contains(filePath, "/") {
		fileName = filePath
	} else {
		fileName = objectStat.Name()
	}

	n, err := c.minioclient.PutObject(context.Background(), bucket, fileName, object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})	if err != nil {
		return err
	}

	fmt.Printf("uploaded %q of size %d\n", filePath, n.Size)
	return nil
}

func (c *Client) DownloadFile(bucket, file string) error {
	reader, err := c.minioclient.GetObject(context.Background(), bucket, file, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	if localFile, err := os.Create(file); err != nil {
		return err
	} else {
		if stat, err := reader.Stat(); err != nil {
			return err
		} else {
			if _, err := io.CopyN(localFile, reader, stat.Size); err != nil {
				return err
			}
		}
	}

	return nil
}
