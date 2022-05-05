package workerpool

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	requestCount     = 10
	concurrencylevel = 2
)

func TestWorkerPool(t *testing.T) {
	service := New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencylevel)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go service.SendRequests(testRequests())

	go service.Run(ctx)

	total := 0
	for r := range service.GetResponses() {
		if r.Result.Id != r.Result.TestName || r.Result.Id == "" || r.Result.TestName == "" {
			t.Fatalf("wrong value %v; expected %v", r.Result.Id, r.Result.TestName)
		}

		total++
	}

	if total != requestCount {
		t.Fatalf("wrong value %v; expected %v", total, requestCount)
	}
}

var execFn = func(ctx context.Context, object testkube.Test, options testkube.ExecutionRequest) (result testkube.Execution, err error) {
	return testkube.Execution{Id: options.Name, TestName: object.Name}, nil
}

func testRequests() []Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution] {
	requests := make([]Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution], requestCount)
	for i := 0; i < requestCount; i++ {
		requests[i] = Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution]{
			Object: testkube.Test{
				Name: fmt.Sprintf("%v", i),
			},
			Options: testkube.ExecutionRequest{
				Name: fmt.Sprintf("%v", i),
			},
			ExecFn: execFn,
		}
	}
	return requests
}
