package mongo

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
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

func TestNewMongoRepository_Executions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
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

	res, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter())
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 3)

	labelSelector := testworkflow.LabelSelector{
		Or: []testworkflow.Label{
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-group2-name", res[0].Name)

	labelSelector = testworkflow.LabelSelector{
		Or: []testworkflow.Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
		},
	}
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-no-group-name", res[0].Name)

	labelSelector = testworkflow.LabelSelector{
		Or: []testworkflow.Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithLabelSelector(&labelSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	labelSelector = testworkflow.LabelSelector{
		Or: []testworkflow.Label{
			{Key: "workflow.labels.testkube.io/group", Exists: boolPtr(false)},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp1")},
			{Key: "workflow.labels.testkube.io/group", Value: strPtr("grp2")},
		},
	}
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithLabelSelector(&labelSelector))
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

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
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

	res, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter())
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 3)

	tagSelector := "my.key1=value1"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-1", res[0].Name)

	tagSelector = "my.key1"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	tagSelector = "my.key1=value3,key2"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-3", res[0].Name)

	tagSelector = "my.key1=value1,key2=value2"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 0)

	tagSelector = "my.key1=value1,my.key1=value3"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithTagSelector(tagSelector))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)
}

func TestNewMongoRepository_GetExecutions_Actor_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-executions-actor-mongo-repository-test")
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
		RunningContext: &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Name:  "user-1",
				Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			},
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
		RunningContext: &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Name:  "user-2",
				Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			},
		},
	}
	if err := repo.Insert(ctx, execution); err != nil {
		t.Fatalf("error inserting execution: %v", err)
	}

	res, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter())
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	actorName := "user-1"
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithActorName(actorName))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-1", res[0].Name)

	actorType := testkube.USER_TestWorkflowRunningContextActorType
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithActorType(actorType))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 2)

	actorName = "user-1"
	actorType = testkube.USER_TestWorkflowRunningContextActorType
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithActorName(actorName).WithActorType(actorType))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 1)
	assert.Equal(t, "test-name-1", res[0].Name)

	actorName = "user-1"
	actorType = testkube.PROGRAM_TestWorkflowRunningContextActorType
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithActorName(actorName).WithActorType(actorType))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 0)

	actorName = "user-3"
	actorType = testkube.USER_TestWorkflowRunningContextActorType
	res, err = repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithActorName(actorName).WithActorType(actorType))
	if err != nil {
		t.Fatalf("error getting executions: %v", err)
	}

	assert.Len(t, res, 0)
}

func TestNewMongoRepository_GetExecutionsSummary_Integration(t *testing.T) {
	test.IntegrationTest(t)
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-executions-summary-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})
	repo := NewMongoRepository(db, false)

	// Insert test data
	execution := testkube.TestWorkflowExecution{
		Id:   "test-id-1",
		Name: "test-name-1",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow-1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name: "test-workflow-1",
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"param1": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
					},
				},
			},
		},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{},
		Reports: []testkube.TestWorkflowReport{
			{
				Summary: &testkube.TestWorkflowReportSummary{
					Passed:   10,
					Failed:   2,
					Skipped:  3,
					Errored:  1,
					Tests:    16,
					Duration: 15000,
				},
			},
		},
	}
	err = repo.Insert(ctx, execution)
	assert.NoError(t, err)

	execution2 := testkube.TestWorkflowExecution{
		Id:   "test-id-1",
		Name: "test-name-2",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow-2",
			Spec: &testkube.TestWorkflowSpec{},
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name: "test-workflow-2",
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"param1": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
					},
				},
			},
		},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{
			"param1": {
				Value: "custom-value",
			},
		},
	}
	err = repo.Insert(ctx, execution2)
	assert.NoError(t, err)

	// Test GetExecutionsSummary
	filter := testworkflow.NewExecutionsFilter().WithName("test-workflow-1")
	result, err := repo.GetExecutionsSummary(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "test-name-1", result[0].Name)
	assert.Equal(t, "default", result[0].ConfigParams["param1"].DefaultValue)
	assert.Len(t, result[0].Reports, 1)
	assert.EqualValues(t, 10, result[0].Reports[0].Summary.Passed)
	assert.EqualValues(t, 2, result[0].Reports[0].Summary.Failed)
	assert.EqualValues(t, 3, result[0].Reports[0].Summary.Skipped)
	assert.EqualValues(t, 1, result[0].Reports[0].Summary.Errored)
	assert.EqualValues(t, 16, result[0].Reports[0].Summary.Tests)
	assert.EqualValues(t, 15000, result[0].Reports[0].Summary.Duration)

	filter = testworkflow.NewExecutionsFilter().WithName("test-workflow-2")
	result, err = repo.GetExecutionsSummary(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "test-name-2", result[0].Name)
	assert.Equal(t, "default", result[0].ConfigParams["param1"].DefaultValue)
	assert.Equal(t, "custom-value", result[0].ConfigParams["param1"].Value)
	assert.Len(t, result[0].Reports, 0)
}

func TestNewMongoRepository_Get_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-get-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)
	execution := testkube.TestWorkflowExecution{
		Id:   "test-id-1",
		Name: "test-name-1",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow-1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name: "test-workflow-1",
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"param1": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
					},
				},
			},
		},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{},
	}
	err = repo.Insert(ctx, execution)
	assert.NoError(t, err)

	result, err := repo.Get(ctx, "test-id-1")
	assert.NoError(t, err)

	assert.Equal(t, execution.Id, result.Id)
	assert.Equal(t, execution.Name, result.Name)
	assert.Equal(t, "default", result.ConfigParams["param1"].DefaultValue)
	assert.Equal(t, false, result.ConfigParams["param1"].Truncated)

	execution2 := testkube.TestWorkflowExecution{
		Id:   "test-id-2",
		Name: "test-name-2",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow-2",
			Spec: &testkube.TestWorkflowSpec{},
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name: "test-workflow-2",
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"param2": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
					},
					"param1": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
						Sensitive: true,
					},
				},
			},
		},
	}
	err = repo.Insert(ctx, execution2)
	assert.NoError(t, err)

	result, err = repo.Get(ctx, "test-id-2")
	assert.NoError(t, err)

	assert.Equal(t, execution2.Id, result.Id)
	assert.Equal(t, execution2.Name, result.Name)
	assert.Equal(t, true, result.ConfigParams["param1"].Sensitive)

	execution3 := testkube.TestWorkflowExecution{
		Id:   "test-id-3",
		Name: "test-name-3",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow-3",
			Spec: &testkube.TestWorkflowSpec{},
		},
		ResolvedWorkflow: &testkube.TestWorkflow{
			Name: "test-workflow-2",
			Spec: &testkube.TestWorkflowSpec{
				Config: map[string]testkube.TestWorkflowParameterSchema{
					"param2": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
					},
					"param1": {
						Default_: &testkube.BoxedString{
							Value: "default",
						},
						Sensitive: true,
					},
				},
			},
		},
		ConfigParams: map[string]testkube.TestWorkflowExecutionConfigValue{},
	}
	err = repo.Insert(ctx, execution3)
	assert.NoError(t, err)

	result, err = repo.Get(ctx, "test-id-3")
	assert.NoError(t, err)

	assert.Equal(t, execution3.Id, result.Id)
	assert.Equal(t, execution3.Name, result.Name)
	assert.Equal(t, true, result.ConfigParams["param1"].Sensitive)
}
