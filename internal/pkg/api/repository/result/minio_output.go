package result

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MinioRepository struct {
	storage             storage.Client
	executionCollection *mongo.Collection
	bucket              string
}

func NewMinioOutputRepository(storageClient storage.Client, executionCollection *mongo.Collection, bucket string) *MinioRepository {
	return &MinioRepository{
		storage:             storageClient,
		executionCollection: executionCollection,
		bucket:              bucket,
	}
}

func (m *MinioRepository) GetOutput(ctx context.Context, id string) (output string, err error) {
	eOutput, err := m.getOutput(id)
	if err != nil {
		return "", err
	}
	return eOutput.Output, err
}

func (m *MinioRepository) getOutput(id string) (ExecutionOutput, error) {
	file, err := m.storage.DownloadFile(m.bucket, id)
	if err != nil {
		return ExecutionOutput{}, err
	}
	var eOutput ExecutionOutput
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&eOutput)
	if err != nil {
		return ExecutionOutput{}, err
	}
	return eOutput, err
}

func (m *MinioRepository) saveOutput(eOutput ExecutionOutput) error {
	data, err := json.Marshal(eOutput)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	m.storage.UploadFile(m.bucket, eOutput.Id, reader, reader.Size())
	return err
}

func (m *MinioRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	eOutput := ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output}
	return m.saveOutput(eOutput)
}

func (m *MinioRepository) UpdateOutput(ctx context.Context, id, output string) error {
	eOutput, err := m.getOutput(id)
	if err != nil {
		return err
	}
	eOutput.Output = output
	return m.saveOutput(eOutput)
}

func (m *MinioRepository) DeleteOutput(ctx context.Context, id string) error {
	return m.storage.DeleteFile(m.bucket, id)
}

func (m *MinioRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
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
		err = m.DeleteOutput(ctx, execution.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
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
		err = m.DeleteOutput(ctx, execution.Id)
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
		err = m.DeleteOutput(ctx, execution.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioRepository) DeleteAllOutput(ctx context.Context) error {
	err := m.storage.DeleteBucket(m.bucket, true)
	if err != nil {
		return err
	}
	return m.storage.CreateBucket(m.bucket)
}
