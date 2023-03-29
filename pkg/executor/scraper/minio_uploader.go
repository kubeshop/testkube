package scraper

import (
	"context"

	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/utils"
)

type MinIOUploader struct {
	Endpoint, AccessKeyID, SecretAccessKey, Region, Token, Bucket string
	Ssl                                                           bool
	client                                                        *minio.Client
}

func NewMinIOLoader(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) (*MinIOUploader, error) {
	l := &MinIOUploader{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
		Token:           token,
		Bucket:          bucket,
		Ssl:             ssl,
	}

	client := minio.NewClient(l.Endpoint, l.AccessKeyID, l.SecretAccessKey, l.Region, l.Token, l.Bucket, l.Ssl)
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
	if err := l.client.SaveFileDirect(ctx, folder, object.Name, object.Data, object.Size); err != nil {
		return errors.Wrapf(err, "error saving file %s", object.Name)
	}

	return nil
}

func ExtractMinIOUploaderMeta(execution testkube.Execution) map[string]any {
	return map[string]any{
		"executionId": execution.Id,
	}
}
