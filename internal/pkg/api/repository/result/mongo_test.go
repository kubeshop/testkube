package result

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/datefilter"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/storage"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/stretchr/testify/require"
)

const (
	mongoDns    = "mongodb://localhost:27017"
	mongoDbName = "testkube"
)

func TestFilters(t *testing.T) {
	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.Coll.Drop(context.TODO())
	assert.NoError(err)

	oneDayAgo := time.Now().Add(-24 * time.Hour)
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	defaultName := "name"
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, time.Now(), []string{"test1", "test2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, time.Now(), []string{"test1", "test2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, time.Now(), []string{"test3", "test4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, time.Now(), []string{"test3", "test4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.SUCCESS_ExecutionStatus, time.Now(), []string{"test1", "test4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, time.Now(), []string{"test1", "test3"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PENDING_ExecutionStatus, time.Now(), []string{"test5", "test6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, oneDayAgo, []string{"test1", "test5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, oneDayAgo, []string{"test1", "test6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, oneDayAgo, []string{"test2", "test4"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, oneDayAgo, []string{"test2", "test5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.SUCCESS_ExecutionStatus, oneDayAgo, []string{"test7", "test8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, oneDayAgo, []string{"test7", "test8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PENDING_ExecutionStatus, oneDayAgo, []string{"test7", "test8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, twoDaysAgo, []string{"test7", "test8"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, twoDaysAgo, []string{"test1", "test2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, twoDaysAgo, []string{"test1", "test2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.ERROR__ExecutionStatus, twoDaysAgo, []string{"test1", "test2"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.SUCCESS_ExecutionStatus, twoDaysAgo, []string{"test3", "test6"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.QUEUED_ExecutionStatus, twoDaysAgo, []string{"test3", "test5"})
	assert.NoError(err)
	err = repository.insertExecutionResult(defaultName, testkube.PENDING_ExecutionStatus, twoDaysAgo, []string{"test4", "test6"})
	assert.NoError(err)

	t.Run("filter with status should return only executions with that status", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithStatus(testkube.ERROR__ExecutionStatus))
		assert.NoError(err)
		assert.Len(executions, 12)
		assert.Equal(*executions[0].ExecutionResult.Status, testkube.ERROR__ExecutionStatus)
	})

	t.Run("filter with status should return only totals with that status", func(t *testing.T) {
		filteredTotals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter().WithStatus(testkube.ERROR__ExecutionStatus))

		assert.NoError(err)
		assert.Equal(int32(12), filteredTotals.Results)
		assert.Equal(int32(12), filteredTotals.Failed)
		assert.Equal(int32(0), filteredTotals.Passed)
		assert.Equal(int32(0), filteredTotals.Queued)
		assert.Equal(int32(0), filteredTotals.Pending)
	})

	t.Run("getting totals without filters should return all the executions", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter())

		assert.NoError(err)
		assert.Equal(int32(21), totals.Results)
		assert.Equal(int32(12), totals.Failed)
		assert.Equal(int32(3), totals.Passed)
		assert.Equal(int32(3), totals.Queued)
		assert.Equal(int32(3), totals.Pending)
	})

	dateFilter := datefilter.NewDateFilter(oneDayAgo.Format(datefilter.DateFormatISO8601), "")
	assert.True(dateFilter.IsStartValid)

	t.Run("filter with startDate should return only executions after that day", func(t *testing.T) {
		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithStartDate(dateFilter.Start))
		assert.NoError(err)
		assert.Len(executions, 14)
		assert.True(executions[0].StartTime.After(dateFilter.Start) || executions[0].StartTime.Equal(dateFilter.Start))
	})

	t.Run("filter with tags should return only filters with given tags", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithTags([]string{"test1", "test2"}))
		assert.NoError(err)
		assert.Len(executions, 5)
	})

	t.Run("getting totals with filter by date start date should return only the results after this date", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter().WithStartDate(dateFilter.Start))

		assert.NoError(err)
		assert.Equal(int32(14), totals.Results)
		assert.Equal(int32(8), totals.Failed)
		assert.Equal(int32(2), totals.Passed)
		assert.Equal(int32(2), totals.Queued)
		assert.Equal(int32(2), totals.Pending)
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
		totals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter().WithEndDate(dateFilter.End))

		assert.NoError(err)
		assert.Equal(int32(7), totals.Results)
		assert.Equal(int32(4), totals.Failed)
		assert.Equal(int32(1), totals.Passed)
		assert.Equal(int32(1), totals.Queued)
		assert.Equal(int32(1), totals.Pending)
	})

	t.Run("filter with script name that doesn't exist should return 0 results", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithScriptName("noneExisting"))
		assert.NoError(err)
		assert.Empty(executions)
	})

	t.Run("getting totals with script name that doesn't exist should return 0 results", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter().WithScriptName("noneExisting"))

		assert.NoError(err)
		assert.Equal(int32(0), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(0), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(0), totals.Pending)
	})

	t.Run("filter with ccombined filter should return corresponding results", func(t *testing.T) {
		filter := NewExecutionsFilter().
			WithStatus(testkube.SUCCESS_ExecutionStatus).
			WithStartDate(twoDaysAgo).
			WithEndDate(oneDayAgo).
			WithScriptName(defaultName)
		executions, err := repository.GetExecutions(context.Background(), filter)
		assert.NoError(err)
		assert.Len(executions, 2)
	})

	t.Run("getting totals with ccombined filter should return corresponding results", func(t *testing.T) {
		filter := NewExecutionsFilter().
			WithStatus(testkube.SUCCESS_ExecutionStatus).
			WithStartDate(twoDaysAgo).
			WithEndDate(oneDayAgo).
			WithScriptName(defaultName)
		totals, err := repository.GetExecutionTotals(context.Background(), filter)

		assert.NoError(err)
		assert.Equal(int32(2), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(2), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(0), totals.Pending)
	})

	name := "someDifferentName"
	err = repository.insertExecutionResult(name, testkube.PENDING_ExecutionStatus, twoDaysAgo, nil)
	assert.NoError(err)

	t.Run("filter with script name should return result only for that script name", func(t *testing.T) {

		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter().WithScriptName(name))
		assert.NoError(err)
		assert.Len(executions, 1)
		assert.Equal(executions[0].ScriptName, name)
	})

	t.Run("getting totals with script name should return result only for that script name", func(t *testing.T) {
		totals, err := repository.GetExecutionTotals(context.Background(), NewExecutionsFilter().WithScriptName(name))

		assert.NoError(err)
		assert.Equal(int32(1), totals.Results)
		assert.Equal(int32(0), totals.Failed)
		assert.Equal(int32(0), totals.Passed)
		assert.Equal(int32(0), totals.Queued)
		assert.Equal(int32(1), totals.Pending)
	})

	t.Run("scripts should be sorted with most recent first", func(t *testing.T) {
		executions, err := repository.GetExecutions(context.Background(), NewExecutionsFilter())
		assert.NoError(err)
		assert.NotEmpty(executions)
		assert.True(executions[0].StartTime.After(executions[len(executions)-1].StartTime), "executions are not sorted with the most recent first")
	})
}

func getRepository() (*MongoRepository, error) {
	db, err := storage.GetMongoDataBase(mongoDns, mongoDbName)
	repository := NewMongoRespository(db)
	return repository, err
}

func (repository *MongoRepository) insertExecutionResult(scriptName string, execStatus testkube.ExecutionStatus, startTime time.Time, tags []string) error {
	return repository.Insert(context.Background(),
		testkube.Execution{
			Id:              rand.Name(),
			ScriptName:      scriptName,
			Name:            "dummyName",
			ScriptType:      "test/curl",
			StartTime:       startTime,
			EndTime:         time.Now(),
			ExecutionResult: &testkube.ExecutionResult{Status: &execStatus},
			Tags:            tags,
		})
}
