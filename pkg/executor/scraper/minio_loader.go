package scraper

import (
	"context"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/utils"
)

type MinIOLoader struct {
	Endpoint, AccessKeyID, SecretAccessKey, Location, Token, Bucket string
	Ssl                                                             bool
	client                                                          *minio.Client
}

func NewMinIOLoader(endpoint, accessKeyID, secretAccessKey, location, token, bucket string, ssl bool) (*MinIOLoader, error) {
	l := &MinIOLoader{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Location:        location,
		Token:           token,
		Bucket:          bucket,
		Ssl:             ssl,
	}

	client := minio.NewClient(l.Endpoint, l.AccessKeyID, l.SecretAccessKey, l.Location, l.Token, l.Bucket, l.Ssl)
	err := client.Connect()
	if err != nil {
		return nil, errors.Errorf("error occured creating minio client: %v", err)
	}
	l.client = client

	return l, nil
}

func (l *MinIOLoader) Load(ctx context.Context, object *Object, meta map[string]any) error {
	client := minio.NewClient(l.Endpoint, l.AccessKeyID, l.SecretAccessKey, l.Location, l.Token, l.Bucket, l.Ssl) // create storage client
	err := client.Connect()
	if err != nil {
		return errors.Errorf("error occured creating minio client: %v", err)
	}

	if err := utils.CheckStringKey(meta, "executionId"); err != nil {
		return err
	}

	folder := meta["executionId"].(string)
	if err := client.SaveFileDirect(ctx, folder, object.Name, object.Data, object.Size); err != nil {
		return errors.Wrapf(err, "error saving file %s", object.Name)
	}

	return nil
}
