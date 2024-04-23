package workerpool

import (
	"context"
	"sync"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Runnable is an interface of runnable objects
type Runnable interface {
	testkube.Test | testkube.TestSuite | testworkflowsv1.TestWorkflow
}

// Requestable is an interface of requestable objects
type Requestable interface {
	testkube.ExecutionRequest | testkube.TestSuiteExecutionRequest | testkube.TestWorkflowExecutionRequest
}

// Returnable is an interface of returnable objects
type Returnable interface {
	testkube.Execution | testkube.TestSuiteExecution | testkube.TestWorkflowExecution
}

// ExecuteFn is a function type for executing runnable and requestable parameters with returnable results
type ExecuteFn[R Runnable, T Requestable, E Returnable] func(ctx context.Context, object R, options T) (result E, err error)

// Request contains request parameters and invocation method
type Request[R Runnable, T Requestable, E Returnable] struct {
	Object  R
	Options T
	ExecFn  ExecuteFn[R, T, E]
}

// Response contains result details
type Response[E Returnable] struct {
	Result E
	Err    error
}

// execute is a method wrapper for ExecFn execution
func (r Request[R, T, E]) execute(ctx context.Context) Response[E] {
	result, err := r.ExecFn(ctx, r.Object, r.Options)
	if err != nil {
		return Response[E]{
			Err: err,
		}
	}

	return Response[E]{
		Result: result,
	}
}

// Service is a worker pool service
type Service[R Runnable, T Requestable, E Returnable] struct {
	concurrencyLevel int
	requests         chan Request[R, T, E]
	responses        chan Response[E]
}

// New is a constructor for worker pool service
func New[R Runnable, T Requestable, E Returnable](concurrencyLevel int) Service[R, T, E] {
	return Service[R, T, E]{
		concurrencyLevel: concurrencyLevel,
		requests:         make(chan Request[R, T, E], concurrencyLevel),
		responses:        make(chan Response[E], concurrencyLevel),
	}
}

// Run is a method to run worker pool
func (s Service[R, T, E]) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < s.concurrencyLevel; i++ {
		wg.Add(1)
		go worker(ctx, &wg, s.requests, s.responses)
	}

	wg.Wait()
	close(s.responses)
}

// GetResponse return reponses of method execution
func (s Service[R, T, E]) GetResponses() <-chan Response[E] {
	return s.responses
}

// SendRequests sends requests to workers
func (s Service[R, T, E]) SendRequests(requests []Request[R, T, E]) {
	for i := range requests {
		s.requests <- requests[i]
	}
	close(s.requests)
}

// worker is a worker pool method
func worker[R Runnable, T Requestable, E Returnable](ctx context.Context, wg *sync.WaitGroup,
	requests <-chan Request[R, T, E], responses chan<- Response[E]) {
	defer wg.Done()
	for {
		select {
		case request, ok := <-requests:
			if !ok {
				return
			}

			responses <- request.execute(ctx)
		case <-ctx.Done():
			return
		}
	}
}
