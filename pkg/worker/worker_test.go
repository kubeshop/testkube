package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestWorker_Run(t *testing.T) {
	t.Run("worker channel pipeline", func(t *testing.T) {
		runner := &RunnerMock{T: t}
		repo := &RepositoryMock{}
		worker := NewWorker(repo, runner)

		execution := testkube.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &testkube.ExecutionResult{}

		executionChan := make(chan testkube.Execution, 2)
		executionChan <- execution

		worker.Run(executionChan)
		time.Sleep(10 * time.Millisecond)
	})
}

func TestWorker_RunExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := &RunnerMock{T: t}
		repo := &RepositoryMock{}
		worker := NewWorker(repo, runner)

		execution := testkube.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &testkube.ExecutionResult{}

		result, err := worker.RunExecution(context.Background(), execution)
		assert.NoError(t, err)
		assert.Equal(t, *result.ExecutionResult.Status, testkube.SUCCESS_ExecutionStatus)
	})

	t.Run("runner error", func(t *testing.T) {
		rerr := fmt.Errorf("runner error")
		runner := &RunnerMock{T: t, Error: rerr}
		repo := &RepositoryMock{}
		worker := NewWorker(repo, runner)

		execution := testkube.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &testkube.ExecutionResult{}

		result, err := worker.RunExecution(context.Background(), execution)
		assert.Error(t, err)
		assert.Equal(t, *result.ExecutionResult.Status, testkube.ERROR__ExecutionStatus)
	})
}

type RunnerMock struct {
	Error       error
	ErrorResult error
	Result      string
	T           *testing.T
}

func (r RunnerMock) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
	result := *execution.ExecutionResult
	status := testkube.SUCCESS_ExecutionStatus
	result.Status = &status
	if r.ErrorResult != nil {
		result.Err(r.Error)
	}
	return result, r.Error
}

type RepositoryMock struct {
	Result    testkube.Execution
	CallCount int
	Error     error
}

// Get gets execution result by id
func (r *RepositoryMock) Get(ctx context.Context, id string) (testkube.Execution, error) {
	r.CallCount++
	return r.Result, r.Error
}

// Insert inserts new execution result
func (r *RepositoryMock) Insert(ctx context.Context, result testkube.Execution) error {
	r.CallCount++
	return r.Error
}

// Update updates execution result
func (r *RepositoryMock) Update(ctx context.Context, result testkube.Execution) error {
	r.CallCount++
	return r.Error
}

//UpdateResult updates only result part of execution
func (r *RepositoryMock) UpdateResult(ctx context.Context, id string, result testkube.ExecutionResult) (err error) {
	r.CallCount++
	return r.Error

}

// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
func (r *RepositoryMock) QueuePull(ctx context.Context) (testkube.Execution, error) {
	r.CallCount++
	return r.Result, r.Error
}
