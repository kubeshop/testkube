//go:build integration

package testresult

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/storage"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/rand"
)

const (
	mongoDns    = "mongodb://localhost:27017"
	mongoDbName = "testkube-test"
)

func TestTestExecutionsMetrics(t *testing.T) {
	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.Coll.Drop(context.TODO())
	assert.NoError(err)

	testName := "example-test"

	err = repository.insertExecutionResult(testName, testkube.FAILED_TestSuiteExecutionStatus, time.Now().Add(48*-time.Hour), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Hour), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(10*-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(10*-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_TestSuiteExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)

	metrics, err := repository.GetTestSuiteMetrics(context.Background(), testName, 100, 100)
	assert.NoError(err)

	t.Run("getting execution metrics for test data", func(t *testing.T) {
		assert.NoError(err)
		assert.Equal(int32(20), metrics.TotalExecutions)
		assert.Equal(int32(5), metrics.FailedExecutions)
		assert.Len(metrics.Executions, 20)
	})

	t.Run("getting pass/fail ratio", func(t *testing.T) {
		assert.Equal(float64(75), metrics.PassFailRatio)
	})

	t.Run("getting percentiles of execution duration", func(t *testing.T) {
		assert.Contains(metrics.ExecutionDurationP50, "1m0")
		assert.Contains(metrics.ExecutionDurationP90, "10m0")
		assert.Contains(metrics.ExecutionDurationP99, "48h0m0s")
	})

	t.Run("limit should limit executions", func(t *testing.T) {
		metrics, err := repository.GetTestSuiteMetrics(context.Background(), testName, 1, 100)
		assert.NoError(err)
		assert.Equal(1, len(metrics.Executions))
	})

	t.Run("filter last n days should limit executions", func(t *testing.T) {
		metrics, err := repository.GetTestSuiteMetrics(context.Background(), testName, 100, 1)
		assert.NoError(err)
		assert.Equal(int32(19), metrics.TotalExecutions)
	})
}

func getRepository() (*MongoRepository, error) {
	db, err := storage.GetMongoDatabase(mongoDns, mongoDbName, storage.TypeMongoDB, false, nil)
	repository := NewMongoRepository(db, true, false)
	return repository, err
}

func (r *MongoRepository) insertExecutionResult(testSuiteName string, execStatus testkube.TestSuiteExecutionStatus, startTime time.Time, labels map[string]string) error {
	return r.Insert(context.Background(),
		testkube.TestSuiteExecution{
			Id:        rand.Name(),
			TestSuite: &testkube.ObjectRef{Namespace: "testkube", Name: testSuiteName},
			Name:      "dummyName",
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime).String(),
			Labels:    labels,
			Status:    &execStatus,
		})
}
