package testworkflow

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestNewMongoRepository_UpdateReport_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)

	execution := testkube.TestWorkflowExecution{
		Id:   "test-id",
		Name: "test-name",
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	summary1 := &testkube.TestWorkflowReportSummary{
		Passed:   1,
		Failed:   2,
		Skipped:  3,
		Errored:  4,
		Tests:    10,
		Duration: 12000,
	}
	report1 := &testkube.TestWorkflowReport{
		Ref:     "test-ref",
		Kind:    "junit",
		File:    "test-file",
		Summary: summary1,
	}
	if err := repo.UpdateReport(ctx, execution.Id, report1); err != nil {
		t.Fatalf("error updating report: %v", err)
	}

	summary2 := &testkube.TestWorkflowReportSummary{
		Passed:   2,
		Failed:   3,
		Skipped:  4,
		Errored:  5,
		Tests:    14,
		Duration: 20000,
	}
	report2 := &testkube.TestWorkflowReport{
		Ref:     "test-ref-2",
		Kind:    "junit",
		File:    "test-file-2",
		Summary: summary2,
	}
	if err := repo.UpdateReport(ctx, execution.Id, report2); err != nil {
		t.Fatalf("error updating report: %v", err)
	}

	fresh, err := repo.Get(ctx, execution.Id)
	if err != nil {
		t.Fatalf("error getting execution: %v", err)
	}

	assert.Equal(t, *report1, fresh.Reports[0])
	assert.Equal(t, *report2, fresh.Reports[1])
}
