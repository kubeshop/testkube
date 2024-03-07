// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

import (
	"context"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
)

var _ OutputRepository = (*MinioRepository)(nil)

const bucketFolder = "testworkflows"

type MinioRepository struct {
	storage storage.Client
	bucket  string
}

func NewMinioOutputRepository(storageClient storage.Client, bucket string) *MinioRepository {
	log.DefaultLogger.Debugw("creating minio workflow output repository", "bucket", bucket)
	return &MinioRepository{
		storage: storageClient,
		bucket:  bucket,
	}
}

// PresignSaveLog builds presigned storage URL to save the output in Cloud
func (m *MinioRepository) PresignSaveLog(ctx context.Context, id, workflowName string) (string, error) {
	return m.storage.PresignUploadFileToBucket(ctx, m.bucket, bucketFolder, id, 24*time.Hour)
}

// PresignReadLog builds presigned storage URL to read the output from Cloud
func (m *MinioRepository) PresignReadLog(ctx context.Context, id, workflowName string) (string, error) {
	return m.storage.PresignDownloadFileFromBucket(ctx, m.bucket, bucketFolder, id, 15*time.Minute)
}

func (m *MinioRepository) SaveLog(ctx context.Context, id, workflowName string, reader io.Reader) error {
	log.DefaultLogger.Debugw("inserting output", "id", id, "workflowName", workflowName)
	return m.storage.UploadFileToBucket(ctx, m.bucket, bucketFolder, id, reader, -1)
}

func (m *MinioRepository) ReadLog(ctx context.Context, id, workflowName string) (io.Reader, error) {
	file, _, err := m.storage.DownloadFileFromBucket(ctx, m.bucket, bucketFolder, id)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (m *MinioRepository) HasLog(ctx context.Context, id, workflowName string) (bool, error) {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, _, err := m.storage.DownloadFileFromBucket(subCtx, m.bucket, bucketFolder, id)
	if err != nil {
		return false, err
	}
	return true, nil
}
