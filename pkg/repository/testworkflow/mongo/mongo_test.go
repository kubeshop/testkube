package mongo_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	mongorepo "github.com/kubeshop/testkube/pkg/repository/testworkflow/mongo"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const databaseName = "test"

func setupRepo(t *testing.T) *mongorepo.MongoRepository {
	t.Helper()

	mongoDBContainer, err := mongodb.Run(context.Background(), "mongo:6")
	if err != nil {
		t.Fatalf("Unable to create mongodb: %v", err)
	}
	mongoUrl, err := mongoDBContainer.ConnectionString(context.Background())
	if err != nil {
		t.Fatalf("Unable to create mongodb connection string: %v", err)
	}
	client, err := mongo.Connect(context.Background(),
		options.Client().
			ApplyURI(mongoUrl))
	if err != nil {
		t.Fatalf("Unable to connect to mongodb: %v", err)
	}

	return mongorepo.NewMongoRepository(client.Database(databaseName), false)
}

func TestGetLatestByTestWorkflow_SortByStatusAt(t *testing.T) {
	tests := map[string]struct {
		testData     []testkube.TestWorkflowExecution
		workflowName string
		expect       *testkube.TestWorkflowExecution
	}{
		"one": {
			testData: []testkube.TestWorkflowExecution{
				{Name: "foo", Workflow: &testkube.TestWorkflow{Name: "bar"}},
			},
			workflowName: "bar",
			expect:       &testkube.TestWorkflowExecution{Name: "foo", Workflow: &testkube.TestWorkflow{Name: "bar"}},
		},
		"last status time": {
			testData: []testkube.TestWorkflowExecution{
				{Name: "foo", Workflow: &testkube.TestWorkflow{Name: "bar"}, StatusAt: time.Now().Add(-time.Hour)},
				{Name: "baz", Workflow: &testkube.TestWorkflow{Name: "bar"}, StatusAt: time.Now()},
			},
			workflowName: "bar",
			expect:       &testkube.TestWorkflowExecution{Name: "foo", Workflow: &testkube.TestWorkflow{Name: "bar"}},
		},
		"use result status if available": {
			testData: []testkube.TestWorkflowExecution{
				{Name: "foo", Workflow: &testkube.TestWorkflow{Name: "bar"}, ScheduledAt: time.Now().Add(-time.Hour)},
				{Name: "baz", Workflow: &testkube.TestWorkflow{Name: "bar"}, ScheduledAt: time.Now()},
				{Name: "qux", Workflow: &testkube.TestWorkflow{Name: "bar"}, Result: &testkube.TestWorkflowResult{StartedAt: time.Now().Add(-2 * time.Hour)}},
			},
			workflowName: "bar",
			expect:       &testkube.TestWorkflowExecution{Name: "qux", Workflow: &testkube.TestWorkflow{Name: "bar"}, Result: &testkube.TestWorkflowResult{StartedAt: time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Millisecond)}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo := setupRepo(t)

			var setupErr error
			for i, e := range test.testData {
				if err := repo.Insert(context.Background(), e); err != nil {
					setupErr = errors.Join(setupErr, fmt.Errorf("insert test data execution %d %q: %w", i, e.Name, err))
				}
			}
			if setupErr != nil {
				t.Fatal(setupErr)
			}

			actual, err := repo.GetLatestByTestWorkflow(context.Background(), test.workflowName, testworkflow.LatestSortByStatusAt)
			if err != nil {
				t.Errorf("error returned from function: %v", err)
			}
			if diff := cmp.Diff(test.expect, actual,
				cmpopts.IgnoreFields(testkube.TestWorkflowExecution{}, "Reports"),
				cmpopts.IgnoreTypes(time.Time{}),
			); diff != "" {
				t.Errorf("actual result (-want +got):\n%s", diff)
			}
		})
	}
}
