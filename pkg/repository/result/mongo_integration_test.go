//go:build integration

package result

import (
	"context"
	"fmt"
	random "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/repository/storage"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/rand"
)

const (
	mongoDns    = "mongodb://localhost:27017"
	mongoDbName = "testkube-test"
)

func TestStorage_Integration(t *testing.T) {
	test.IntegrationTest(t)
	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.ResultsColl.Drop(context.TODO())
	assert.NoError(err)

	oneDayAgo := time.Now().Add(-24 * time.Hour)
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	defaultName := "name"
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, time.Now(), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, time.Now(), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, time.Now(), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, time.Now(), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PASSED_ExecutionStatus, time.Now(), map[string]string{"key1": "value1", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, time.Now(), map[string]string{"key1": "value1", "key3": "value3"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.RUNNING_ExecutionStatus, time.Now(), map[string]string{"key5": "value5", "key6": "value6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, oneDayAgo, map[string]string{"key1": "value1", "key5": "value5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, oneDayAgo, map[string]string{"key1": "value1", "key6": "value6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, oneDayAgo, map[string]string{"key2": "value2", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, oneDayAgo, map[string]string{"key2": "value2", "key5": "value5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PASSED_ExecutionStatus, oneDayAgo, map[string]string{"key7": "value7", "key8": "value8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, oneDayAgo, map[string]string{"key7": "value7", "key8": "value8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.RUNNING_ExecutionStatus, oneDayAgo, map[string]string{"key7": "value7", "key8": "value8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, twoDaysAgo, map[string]string{"key7": "value7", "key8": "value8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, twoDaysAgo, map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, twoDaysAgo, map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.FAILED_ExecutionStatus, twoDaysAgo, map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PASSED_ExecutionStatus, twoDaysAgo, map[string]string{"key3": "value3", "key6": "value6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, twoDaysAgo, map[string]string{"key3": "value3", "key5": "value5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.RUNNING_ExecutionStatus, twoDaysAgo, map[string]string{"key4": "value4", "key6": "value6"})
	assert.NoError(err)

	numberOfLabels := 8

	t.Run("filter with status should return only executions with that status", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithStatus(string(testkube.FAILED_ExecutionStatus)))
		assert.NoError(err)
		assert.Len(executions, 12)
		assert.Equal(*executions[0].ExecutionResult.Status, testkube.FAILED_ExecutionStatus)
	})

	t.Run("filter with different statuses should return only executions with those statuses", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithStatus(
			string(testkube.FAILED_ExecutionStatus)+","+string(testkube.PASSED_ExecutionStatus)))
		assert.NoError(err)
		assert.Len(executions, 15)
	})

	t.Run("filter with status should return only totals with that status", func(t *testing.T) {
		filteredTotals, err := repository.GetExecutionTotals(context.Background(), false, NewExecutionsFilter().WithStatus(string(testkube.FAILED_ExecutionStatus)))

		assert.NoError(err)
		assert.Equal(int32(12), filteredTotals.Results)
		assert.Equal(int32(12), filteredTotals.Failed)
		assert.Equal(int32(0), filteredTotals.Passed)
		assert.Equal(int32(0), filteredTotals.Queued)
		assert.Equal(int32(0), filteredTotals.Running)
	})

	t.Run("getting totals without filters should return all the executions", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), false)

		assert.NoError(err)
		assert.Equal(int32(21), totals.Results)
		assert.Equal(int32(12), totals.Failed)
		assert.Equal(int32(3), totals.Passed)
		assert.Equal(int32(3), totals.Queued)
		assert.Equal(int32(3), totals.Running)
	})

	dateFilter := datefilter.NewDateFilter(oneDayAgo.Format(datefilter.DateFormatISO8601), "")
	assert.True(dateFilter.IsStartValid)

	t.Run("filter with startDate should return only executions after that day", func(t *testing.T) {
		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithStartDate(dateFilter.Start))
		assert.NoError(err)
		assert.Len(executions, 14)
		assert.True(executions[0].StartTime.After(dateFilter.Start) || executions[0].StartTime.Equal(dateFilter.Start))
	})

	t.Run("filter with labels should return only filters with given labels", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithSelector("key1=value1,key2=value2"))
		assert.NoError(err)
		assert.Len(executions, 5)
	})

	t.Run("filter with labels should return only filters with existing labels", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithSelector("key1"))
		assert.NoError(err)
		assert.Len(executions, 9)
	})

	t.Run("getting totals with filter by date start date should return only the results after this date", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), false, NewExecutionsFilter().WithStartDate(dateFilter.Start))

		assert.NoError(err)
		assert.Equal(int32(14), totals.Results)
		assert.Equal(int32(8), totals.Failed)
		assert.Equal(int32(2), totals.Passed)
		assert.Equal(int32(2), totals.Queued)
		assert.Equal(int32(2), totals.Running)
	})

	dateFilter = datefilter.NewDateFilter("", oneDayAgo.Format(datefilter.DateFormatISO8601))
	assert.True(dateFilter.IsEndValid)

	t.Run("filter with endDate should return only executions before that day", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithEndDate(dateFilter.End))
		assert.NoError(err)
		assert.Len(executions, 7)
		assert.True(executions[0].StartTime.Before(dateFilter.End) || executions[0].StartTime.Equal(dateFilter.End))
	})

	t.Run("getting totals with filter by date start date should return only the results before this date", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), false, NewExecutionsFilter().WithEndDate(dateFilter.End))

		assert.NoError(err)
		assert.Equal(int32(7), totals.Results)
		assert.Equal(int32(4), totals.Failed)
		assert.Equal(int32(1), totals.Passed)
		assert.Equal(int32(1), totals.Queued)
		assert.Equal(int32(1), totals.Running)
	})

	t.Run("filter with test name that doesn't exist should return 0 results", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithTestName("noneExisting"))
		assert.NoError(err)
		assert.Empty(executions)
	})

	t.Run("getting totals with test name that doesn't exist should return 0 results", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), false, NewExecutionsFilter().WithTestName("noneExisting"))

		assert.NoError(err)
		assert.Equal(int32(0), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(0), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(0), totals.Running)
	})

	t.Run("filter with ccombined filter should return corresponding results", func(t *testing.T) {
		filter := NewExecutionsFilter().
			WithStatus(string(testkube.PASSED_ExecutionStatus)).
			WithStartDate(twoDaysAgo).
			WithEndDate(oneDayAgo).
			WithTestName(defaultName)

		executions, err := repository.GetExecutions(context.Background(), filter)

		assert.NoError(err)
		assert.Len(executions, 2)
	})

	t.Run("getting totals with ccombined filter should return corresponding results", func(t *testing.T) {
		filter := NewExecutionsFilter().
			WithStatus(string(testkube.PASSED_ExecutionStatus)).
			WithStartDate(twoDaysAgo).
			WithEndDate(oneDayAgo).
			WithTestName(defaultName)
		totals, err := repository.GetExecutionTotals(context.Background(), false, filter)

		assert.NoError(err)
		assert.Equal(int32(2), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(2), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(0), totals.Running)
	})

	name := "someDifferentName"
	err = repository.insertExecutionResult(name, testkube.RUNNING_ExecutionStatus, twoDaysAgo, nil)
	assert.NoError(err)

	t.Run("filter with test name should return result only for that test name", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithTestName(name))
		assert.NoError(err)
		assert.Len(executions, 1)
		assert.Equal(executions[0].TestName, name)
	})

	t.Run("getting totals with test name should return result only for that test name", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), false, NewExecutionsFilter().WithTestName(name))

		assert.NoError(err)
		assert.Equal(int32(1), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(0), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(1), totals.Running)
	})

	t.Run("test executions should be sorted with most recent first", func(t *testing.T) {
		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter())
		assert.NoError(err)
		assert.NotEmpty(executions)
		assert.True(executions[0].StartTime.After(executions[len(executions)-1].StartTime), "executions are not sorted with the most recent first")
	})

	t.Run("getting labels should return all available labels", func(t *testing.T) {
		labels, err := repository.GetLabels(context.Background())
		assert.NoError(err)
		assert.Len(labels, numberOfLabels)
	})

}

func TestLabels_Integration(t *testing.T) {
	test.IntegrationTest(t)
	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.ResultsColl.Drop(context.TODO())
	assert.NoError(err)

	t.Run("getting labels when there are no labels should return empty map", func(t *testing.T) {
		labels, err := repository.GetLabels(context.Background())
		assert.NoError(err)
		assert.Len(labels, 0)
	})
}

func TestTestExecutionsMetrics_Integration(t *testing.T) {
	test.IntegrationTest(t)
	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.ResultsColl.Drop(context.TODO())
	assert.NoError(err)

	testName := "example-test"

	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now().Add(48*-time.Hour), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Hour), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(10*-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(10*-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(testName, testkube.PASSED_ExecutionStatus, time.Now().Add(-time.Minute), map[string]string{"key3": "value3", "key4": "value4"})
	assert.NoError(err)

	metrics, err := repository.GetTestMetrics(context.Background(), testName, 100, 100)
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
		metrics, err := repository.GetTestMetrics(context.Background(), testName, 1, 100)
		assert.NoError(err)
		assert.Equal(1, len(metrics.Executions))
	})

	t.Run("filter last n days should limit executions", func(t *testing.T) {
		metrics, err := repository.GetTestMetrics(context.Background(), testName, 100, 1)
		assert.NoError(err)
		assert.Equal(int32(19), metrics.TotalExecutions)
	})
}

func getRepository() (*MongoRepository, error) {
	db, err := storage.GetMongoDatabase(mongoDns, mongoDbName, storage.TypeMongoDB, false, nil)
	repository := NewMongoRepository(db, true, false)
	return repository, err
}

func (r *MongoRepository) insertExecutionResult(testName string, execStatus testkube.ExecutionStatus, startTime time.Time, labels map[string]string) error {
	return r.Insert(context.Background(),
		testkube.Execution{
			Id:              rand.Name(),
			TestName:        testName,
			Name:            "dummyName",
			TestType:        "test/curl",
			StartTime:       startTime,
			EndTime:         time.Now(),
			Duration:        time.Since(startTime).String(),
			ExecutionResult: &testkube.ExecutionResult{Status: &execStatus},
			Labels:          labels,
		})
}

func TestUpdateOutput_Integration(t *testing.T) {
	test.IntegrationTest(t)

	repository, err := getRepository()
	assert.NoError(t, err)

	testName := "example-test"
	executionID := "example-execution"

	err = repository.insertExecutionResult(testName, testkube.FAILED_ExecutionStatus, time.Now(), map[string]string{"key1": "value1", "key2": "value2"})
	assert.NoError(t, err)

	randomString := func(len int) string {
		bytes := make([]byte, len)
		for i := 0; i < len; i++ {
			bytes[i] = byte(65 + random.Intn(25))
		}
		return string(bytes)
	}

	t.Run("valid input", func(t *testing.T) {
		result := testkube.Execution{
			Id:       executionID,
			Name:     executionID,
			TestName: testName,
			ExecutionResult: &testkube.ExecutionResult{
				Output: "",
			},
		}
		err = repository.UpdateResult(context.Background(), testName, result)
		assert.NoError(t, err)
	})
	t.Run("output bigger than MongoDB document limit should not throw error", func(t *testing.T) {
		result := testkube.Execution{
			Id:       executionID,
			Name:     executionID,
			TestName: testName,
			ExecutionResult: &testkube.ExecutionResult{
				Output: randomString(OutputMaxSize),
			},
		}
		err = repository.UpdateResult(context.Background(), testName, result)
		assert.NoError(t, err)
	})
	t.Run("too many steps", func(t *testing.T) {
		result := testkube.Execution{
			Id:       executionID,
			Name:     executionID,
			TestName: testName,
			ExecutionResult: &testkube.ExecutionResult{
				Output: randomString(OutputMaxSize),
				Steps:  []testkube.ExecutionStepResult{},
			},
		}
		for i := 0; i < StepMaxCount; i++ {
			result.ExecutionResult.Steps = append(result.ExecutionResult.Steps, testkube.ExecutionStepResult{
				Name: fmt.Sprintf("step-%d", i),
			})
		}

		err = repository.UpdateResult(context.Background(), testName, result)
		assert.NoError(t, err)
	})
}
