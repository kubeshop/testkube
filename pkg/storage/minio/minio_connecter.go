package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type Connecter struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	ssl             bool
	region          string
	token           string
	bucket          string
	log             *zap.SugaredLogger
	minioClient     *minio.Client
}

// NewConnecter creates new MinioSubscriber which will send data to local MinIO bucket
func NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool, log *zap.SugaredLogger) *Connecter {
	c := &Connecter{
		region:          region,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		token:           token,
		ssl:             ssl,
		bucket:          bucket,
		endpoint:        endpoint,
		log:             log,
	}

	return c
}

// GetClient returns minio client
func (c *Connecter) GetClient() (*minio.Client, error) {
	if c.minioClient == nil {
		var err error
		c.minioClient, err = c.Connect()
		if err != nil {
			return nil, err
		}
	}
	return c.minioClient, nil
}

// Connect connects to MinIO server
func (c *Connecter) Connect() (*minio.Client, error) {
	creds := credentials.NewIAM("")
	c.log.Debugw("connecting to minio",
		"endpoint", c.endpoint,
		"accessKeyID", c.accessKeyID,
		"region", c.region,
		"token", c.token,
		"bucket", c.bucket,
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
	mclient, err := minio.New(c.endpoint, opts)
	if err != nil {
		c.log.Errorw("error connecting to minio", "error", err)
	}
	return mclient, err
}

// Disconnect disconnects from MinIO server
func (c *Connecter) Disconnect() error {
	c.minioClient = nil
	return nil
}
