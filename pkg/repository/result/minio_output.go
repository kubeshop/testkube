package result

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

var _ OutputRepository = (*MinioRepository)(nil)

type MinioRepository struct {
	storage             storage.Client
	executionCollection *mongo.Collection
	bucket              string
}

func NewMinioOutputRepository(storageClient storage.Client, executionCollection *mongo.Collection, bucket string) *MinioRepository {
	log.DefaultLogger.Debugw("creating minio output repository", "bucket", bucket)
	return &MinioRepository{
		storage:             storageClient,
		executionCollection: executionCollection,
		bucket:              bucket,
	}
}

func (m *MinioRepository) GetOutput(ctx context.Context, id, testName, testSuiteName string) (output string, err error) {
	eOutput, err := m.getOutput(ctx, id)
	if err != nil {
		return "", err
	}
	return eOutput.Output, err
}

func (m *MinioRepository) getOutput(ctx context.Context, id string) (ExecutionOutput, error) {
	file, _, err := m.storage.DownloadFileFromBucket(ctx, m.bucket, "", id)
	if err != nil && err == minio.ErrArtifactsNotFound {
		log.DefaultLogger.Infow("output not found in minio", "id", id)
		return ExecutionOutput{}, nil
	}
	if err != nil {
		return ExecutionOutput{}, fmt.Errorf("error downloading output logs from minio: %w", err)
	}
	var eOutput ExecutionOutput
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&eOutput)
	if err != nil {
		return ExecutionOutput{}, err
	}
	return eOutput, err
}

func (m *MinioRepository) saveOutput(ctx context.Context, eOutput ExecutionOutput) error {
	data, err := json.Marshal(eOutput)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	err = m.storage.UploadFileToBucket(ctx, m.bucket, "", eOutput.Id, reader, reader.Size())
	return err
}

func (m *MinioRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	log.DefaultLogger.Debugw("inserting output", "id", id, "testName", testName, "testSuiteName", testSuiteName)
	eOutput := ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output}
	return m.saveOutput(ctx, eOutput)
}

func (m *MinioRepository) UpdateOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	log.DefaultLogger.Debugw("updating output", "id", id)
	eOutput, err := m.getOutput(ctx, id)
	if err != nil {
		return err
	}
	eOutput.Output = output
	return m.saveOutput(ctx, eOutput)
}

func (m *MinioRepository) DeleteOutput(ctx context.Context, id, testName, testSuiteName string) error {
	log.DefaultLogger.Debugw("deleting output", "id", id)
	return m.storage.DeleteFileFromBucket(ctx, m.bucket, "", id)
}

func (m *MinioRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
	log.DefaultLogger.Debugw("deleting output by test", "testName", testName)
	var executions []testkube.Execution
	cursor, err := m.executionCollection.Find(ctx, bson.M{"testname": testName})
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		log.DefaultLogger.Debugw("deleting output for execution", "execution", execution)
		err = m.DeleteOutput(ctx, execution.Id, testName, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
	log.DefaultLogger.Debugw("deleting output for tests", "testNames", testNames)
	for _, testName := range testNames {
		err := m.DeleteOutputByTest(ctx, testName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error {
	var executions []testkube.Execution
	cursor, err := m.executionCollection.Find(ctx, bson.M{"testsuitename": testSuiteName})
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		err = m.DeleteOutput(ctx, execution.Id, "", testSuiteName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error {
	for _, testSuiteName := range testSuiteNames {
		err := m.DeleteOutputByTestSuite(ctx, testSuiteName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MinioRepository) DeleteOutputForAllTestSuite(ctx context.Context) error {
	var executions []testkube.Execution
	cursor, err := m.executionCollection.Find(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		err = m.DeleteOutput(ctx, execution.Id, "", "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteAllOutput(ctx context.Context) error {
	err := m.storage.DeleteBucket(ctx, m.bucket, true)
	if err != nil {
		return err
	}
	return m.storage.CreateBucket(ctx, m.bucket)
}

func (m *MinioRepository) StreamOutput(ctx context.Context, executionID, testName, testSuiteName string) (reader io.Reader, err error) {
	file, _, err := m.storage.DownloadFileFromBucket(ctx, m.bucket, "", executionID)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (m *MinioRepository) GetOutputSize(ctx context.Context, executionID, testName, testSuiteName string) (size int, err error) {
	//TODO: improve with minio client
	stream, err := m.StreamOutput(ctx, executionID, testName, testSuiteName)
	if err != nil {
		return 0, err
	}
	const bufferSize = 1024
	buf := make([]byte, bufferSize)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		size += n
	}
	return size, nil
}
