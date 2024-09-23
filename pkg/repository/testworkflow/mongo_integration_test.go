package testworkflow

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var (
	cfg, _ = config.Get()
)

func TestNewMongoRepository_UpdateReport_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.APIMongoDSN))
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

func TestNewMongoRepository_Executions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.APIMongoDSN))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-executions-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)

	execution := testkube.TestWorkflowExecution{
		Id:   "test-id",
		Name: "test-name",
		Workflow: &testkube.TestWorkflow{
			Name: "test-name",
			Labels: map[string]string{
				"workflow.labels.testkube.io/group": "grp1",
			},
			Spec: &testkube.TestWorkflowSpec{},
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	execution = testkube.TestWorkflowExecution{
		Id:   "test-no-group",
		Name: "test-no-group-name",
		Workflow: &testkube.TestWorkflow{
			Name:   "test-no-group-name",
			Labels: map[string]string{},
			Spec:   &testkube.TestWorkflowSpec{},
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}
	execution = testkube.TestWorkflowExecution{
		Id:   "test-group2-id",
		Name: "test-group2-name",
		Workflow: &testkube.TestWorkflow{
			Name: "test-group2-name",
			Labels: map[string]string{
				"workflow.labels.testkube.io/group": "grp2",
			},
			Spec: &testkube.TestWorkflowSpec{},
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	res, err := repo.GetExecutions(ctx, NewExecutionsFilter())
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 3)

	labelSelector := LabelSelector{
		Or: []Label{
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-group2-name", res[0].Name)

	labelSelector = LabelSelector{
		Or: []Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
		},
	}
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-no-group-name", res[0].Name)

	labelSelector = LabelSelector{
		Or: []Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	labelSelector = LabelSelector{
		Or: []Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp1")},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 3)
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func TestNewMongoRepository_GetExecutions_Tags_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.APIMongoDSN))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-executions-tags-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)

	execution := testkube.TestWorkflowExecution{
		Id:   "test-id-1",
		Name: "test-name-1",
		Workflow: &testkube.TestWorkflow{
			Name: "test-name-1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Tags: map[string]string{
			"my.key1": "value1",
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	execution = testkube.TestWorkflowExecution{
		Id:   "test-id-2",
		Name: "test-name-2",
		Workflow: &testkube.TestWorkflow{
			Name: "test-name-2",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Tags: map[string]string{
			"key2": "value2",
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	execution = testkube.TestWorkflowExecution{
		Id:   "test-id-3",
		Name: "test-name-3",
		Workflow: &testkube.TestWorkflow{
			Name: "test-name-3",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Tags: map[string]string{
			"my.key1": "value3",
			"key2":    "",
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	res, err := repo.GetExecutions(ctx, NewExecutionsFilter())
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 3)

	tagSelector := "my.key1=value1"
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-1", res[0].Name)

	tagSelector = "my.key1"
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	tagSelector = "my.key1=value3,key2"
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-3", res[0].Name)

	tagSelector = "my.key1=value1,key2=value2"
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 0)

	tagSelector = "my.key1=value1,my.key1=value3"
	res, err = repo.GetExecutions(ctx, NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)
}
