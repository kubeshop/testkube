package minio

import (
	"context"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/bufferedstream"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ testworkflow.OutputRepository = (*MinioRepository)(nil)

const bucketFolder = "testworkflows"

type MinioRepository struct {
	storage             storage.Client
	executionCollection *mongo.Collection
	bucket              string
}

func NewMinioOutputRepository(storageClient storage.Client, executionCollection *mongo.Collection, bucket string) *MinioRepository {
	log.DefaultLogger.Debugw("creating minio workflow output repository", "bucket", bucket)
	return &MinioRepository{
		storage:             storageClient,
		executionCollection: executionCollection,
		bucket:              bucket,
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
	buffer, err := bufferedstream.NewBufferedStream("", "log", reader)
	if err != nil {
		return nil
	}
	defer buffer.Cleanup()
	return m.storage.UploadFileToBucket(ctx, m.bucket, bucketFolder, id, buffer, int64(buffer.Len()))
}

func (m *MinioRepository) ReadLog(ctx context.Context, id, workflowName string) (io.ReadCloser, error) {
	file, _, err := m.storage.DownloadFileFromBucket(ctx, m.bucket, bucketFolder, id)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(file), nil
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

func (m *MinioRepository) DeleteOutputByTestWorkflow(ctx context.Context, testWorkflowName string) error {
	log.DefaultLogger.Debugw("deleting output by testWorkflowName", "testWorkflowName", testWorkflowName)
	var executions []testkube.TestWorkflowExecution
	//TODO
	cursor, err := m.executionCollection.Find(ctx, bson.M{"testworkflow.name": testWorkflowName})
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		log.DefaultLogger.Debugw("deleting output for execution", "execution", execution)
		err = m.DeleteOutput(ctx, execution.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutputForTestWorkflows(ctx context.Context, workflowNames []string) error {
	log.DefaultLogger.Debugw("deleting output for testWorkflows", "testWorkflowNames", workflowNames)
	for _, testName := range workflowNames {
		err := m.DeleteOutputByTestWorkflow(ctx, testName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutput(ctx context.Context, id string) error {
	log.DefaultLogger.Debugw("deleting test workflow output", "id", id)
	return m.storage.DeleteFileFromBucket(ctx, m.bucket, bucketFolder, id)
}
