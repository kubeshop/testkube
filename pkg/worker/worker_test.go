package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeshop/kubtest/pkg/api/v1/kubtest"
	"github.com/stretchr/testify/assert"
)

func TestWorker_Run(t *testing.T) {
	t.Run("worker channel pipeline", func(t *testing.T) {
		runner := &RunnerMock{T: t}
		repo := &RepositoryMock{}
		worker := NewWorker(repo, runner)

		execution := kubtest.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &kubtest.ExecutionResult{}

		executionChan := make(chan kubtest.Execution, 2)
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

		execution := kubtest.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &kubtest.ExecutionResult{}

		result, err := worker.RunExecution(context.Background(), execution)
		assert.NoError(t, err)
		assert.Equal(t, *result.ExecutionResult.Status, kubtest.SUCCESS_ExecutionStatus)
	})

	t.Run("runner error", func(t *testing.T) {
		rerr := fmt.Errorf("runner error")
		runner := &RunnerMock{T: t, Error: rerr}
		repo := &RepositoryMock{}
		worker := NewWorker(repo, runner)

		execution := kubtest.NewExecutionWithID("1", "test", "execution-1")
		execution.ExecutionResult = &kubtest.ExecutionResult{}

		result, err := worker.RunExecution(context.Background(), execution)
		assert.Error(t, err)
		assert.Equal(t, *result.ExecutionResult.Status, kubtest.ERROR__ExecutionStatus)
	})
}

type RunnerMock struct {
	Error  error
	Result string
	T      *testing.T
}

func (r RunnerMock) Run(execution kubtest.Execution) kubtest.ExecutionResult {
	result := *execution.ExecutionResult
	status := kubtest.SUCCESS_ExecutionStatus
	result.Status = &status
	if r.Error != nil {
		result.Err(r.Error)
	}
	return result
}

type RepositoryMock struct {
	Result    kubtest.Execution
	CallCount int
	Error     error
}

// Get gets execution result by id
func (r *RepositoryMock) Get(ctx context.Context, id string) (kubtest.Execution, error) {
	r.CallCount++
	return r.Result, r.Error
}

// Insert inserts new execution result
func (r *RepositoryMock) Insert(ctx context.Context, result kubtest.Execution) error {
	r.CallCount++
	return r.Error
}

// Update updates execution result
func (r *RepositoryMock) Update(ctx context.Context, result kubtest.Execution) error {
	r.CallCount++
	return r.Error
}

//UpdateResult updates only result part of execution
func (r *RepositoryMock) UpdateResult(ctx context.Context, id string, result kubtest.ExecutionResult) (err error) {
	r.CallCount++
	return r.Error

}

// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
func (r *RepositoryMock) QueuePull(ctx context.Context) (kubtest.Execution, error) {
	r.CallCount++
	return r.Result, r.Error
}
