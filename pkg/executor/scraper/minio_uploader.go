package scraper

import (
	"context"
	coreminio "github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/utils"
)

type MinIOUploader struct {
	Endpoint, AccessKeyID, SecretAccessKey, Location, Token, Bucket string
	Ssl                                                             bool
	client                                                          *minio.Client
}

func NewMinIOLoader(endpoint, accessKeyID, secretAccessKey, location, token, bucket string, ssl bool) (*MinIOUploader, error) {
	l := &MinIOUploader{
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

func (l *MinIOUploader) Upload(ctx context.Context, object *Object, meta map[string]any) error {
	folder, err := utils.GetStringKey(meta, "executionId")
	if err != nil {
		return err
	}

	log.DefaultLogger.Infow("MinIO loader is uploading file", "file", object.Name, "folder", folder, "size", object.Size)
	opts := coreminio.PutObjectOptions{DisableMultipart: true, UserMetadata: map[string]string{"X-Amz-Meta-Snowball-Auto-Extract": "true"}}
	if err := l.client.SaveFileDirect(ctx, folder, object.Name, object.Data, object.Size, opts); err != nil {
		return errors.Wrapf(err, "error saving file %s", object.Name)
	}

	return nil
}

func ExtractMinIOUploaderMeta(execution testkube.Execution) map[string]any {
	return map[string]any{
		"executionId": execution.Id,
	}
}
