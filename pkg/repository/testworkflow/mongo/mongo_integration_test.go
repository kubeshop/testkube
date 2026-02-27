package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
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

func TestNewMongoRepository_GetFinished_PreservesFilterAnd_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-getfinished-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)

	finishedAt := time.Now()
	passed := testkube.PASSED_TestWorkflowStatus
	failed := testkube.FAILED_TestWorkflowStatus

	// Execution A: finished passed, env=prod - should be returned with tag filter env=prod
	execA := testkube.TestWorkflowExecution{
		Id:   "finished-prod-passed",
		Name: "finished-prod-passed",
		Workflow: &testkube.TestWorkflow{
			Name: "wf1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Result: &testkube.TestWorkflowResult{
			Status:     &passed,
			FinishedAt: finishedAt,
		},
		Tags: map[string]string{"env": "prod"},
	}
	assert.NoError(t, repo.Insert(ctx, execA))

	// Execution B: finished failed, env=staging - should NOT be returned with tag filter env=prod
	execB := testkube.TestWorkflowExecution{
		Id:   "finished-staging-failed",
		Name: "finished-staging-failed",
		Workflow: &testkube.TestWorkflow{
			Name: "wf1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Result: &testkube.TestWorkflowResult{
			Status:     &failed,
			FinishedAt: finishedAt,
		},
		Tags: map[string]string{"env": "staging"},
	}
	assert.NoError(t, repo.Insert(ctx, execB))

	// Execution C: finished passed, env=prod, but SilentMode.Health=true - should be excluded by GetFinished
	execC := testkube.TestWorkflowExecution{
		Id:   "finished-prod-silent",
		Name: "finished-prod-silent",
		Workflow: &testkube.TestWorkflow{
			Name: "wf1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Result: &testkube.TestWorkflowResult{
			Status:     &passed,
			FinishedAt: finishedAt,
		},
		Tags:       map[string]string{"env": "prod"},
		SilentMode: &testkube.SilentMode{Health: true},
	}
	assert.NoError(t, repo.Insert(ctx, execC))

	// Execution D: finished passed, env=prod - should be returned with tag filter env=prod
	execD := testkube.TestWorkflowExecution{
		Id:   "finished-prod-passed-2",
		Name: "finished-prod-passed-2",
		Workflow: &testkube.TestWorkflow{
			Name: "wf1",
			Spec: &testkube.TestWorkflowSpec{},
		},
		Result: &testkube.TestWorkflowResult{
			Status:     &passed,
			FinishedAt: finishedAt,
		},
		Tags: map[string]string{"env": "prod"},
	}
	assert.NoError(t, repo.Insert(ctx, execD))

	filter := testworkflow.NewExecutionsFilter().WithName("wf1").WithTagSelector("env=prod")
	executions, err := repo.GetFinished(ctx, filter)
	assert.NoError(t, err)

	assert.Len(t, executions, 2, "expected exactly 2 executions (A and D)")
	ids := map[string]struct{}{executions[0].Id: {}, executions[1].Id: {}}
	assert.Contains(t, ids, "finished-prod-passed")
	assert.Contains(t, ids, "finished-prod-passed-2")
	assert.NotContains(t, ids, "finished-staging-failed", "execution with env=staging must be excluded by tag filter")
	assert.NotContains(t, ids, "finished-prod-silent", "execution with SilentMode.Health=true must be excluded")
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

func TestNewMongoRepository_SoftDelete_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("testworkflow-soft-delete-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db, false)

	t.Run("DeleteByTestWorkflow sets deletedat and hides from queries", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Id:   "sd-1",
			Name: "sd-name-1",
			Workflow: &testkube.TestWorkflow{
				Name: "sd-workflow",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		require.NoError(t, repo.Insert(ctx, execution))

		// Verify visible before delete
		_, err := repo.Get(ctx, "sd-1")
		require.NoError(t, err)

		// Soft-delete
		err = repo.DeleteByTestWorkflow(ctx, "sd-workflow")
		require.NoError(t, err)

		// Verify hidden from Get
		_, err = repo.Get(ctx, "sd-1")
		assert.ErrorIs(t, err, mongo.ErrNoDocuments)

		// Verify hidden from GetExecutions
		results, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithName("sd-workflow"))
		require.NoError(t, err)
		assert.Empty(t, results)

		// Verify document still exists in collection with deletedat set
		var raw bson.M
		err = repo.Coll.FindOne(ctx, bson.M{"id": "sd-1"}).Decode(&raw)
		require.NoError(t, err)
		assert.NotNil(t, raw["deletedat"], "deletedat should be set on soft-deleted document")
	})

	t.Run("DeleteAll soft-deletes all documents", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Id:   "sd-2",
			Name: "sd-name-2",
			Workflow: &testkube.TestWorkflow{
				Name: "sd-workflow-2",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		require.NoError(t, repo.Insert(ctx, execution))

		err := repo.DeleteAll(ctx)
		require.NoError(t, err)

		// Verify hidden
		_, err = repo.Get(ctx, "sd-2")
		assert.ErrorIs(t, err, mongo.ErrNoDocuments)

		// Verify document exists with deletedat
		var raw bson.M
		err = repo.Coll.FindOne(ctx, bson.M{"id": "sd-2"}).Decode(&raw)
		require.NoError(t, err)
		assert.NotNil(t, raw["deletedat"])
	})

	t.Run("DeleteByTestWorkflows soft-deletes matching workflows", func(t *testing.T) {
		// Clean the collection for this sub-test
		repo.Coll.Drop(ctx)

		exec1 := testkube.TestWorkflowExecution{
			Id:   "sd-3a",
			Name: "sd-name-3a",
			Workflow: &testkube.TestWorkflow{
				Name: "wf-a",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		exec2 := testkube.TestWorkflowExecution{
			Id:   "sd-3b",
			Name: "sd-name-3b",
			Workflow: &testkube.TestWorkflow{
				Name: "wf-b",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		exec3 := testkube.TestWorkflowExecution{
			Id:   "sd-3c",
			Name: "sd-name-3c",
			Workflow: &testkube.TestWorkflow{
				Name: "wf-c",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		require.NoError(t, repo.Insert(ctx, exec1))
		require.NoError(t, repo.Insert(ctx, exec2))
		require.NoError(t, repo.Insert(ctx, exec3))

		// Soft-delete wf-a and wf-b
		err := repo.DeleteByTestWorkflows(ctx, []string{"wf-a", "wf-b"})
		require.NoError(t, err)

		// wf-a and wf-b should be hidden
		_, err = repo.Get(ctx, "sd-3a")
		assert.ErrorIs(t, err, mongo.ErrNoDocuments)
		_, err = repo.Get(ctx, "sd-3b")
		assert.ErrorIs(t, err, mongo.ErrNoDocuments)

		// wf-c should still be visible
		result, err := repo.Get(ctx, "sd-3c")
		require.NoError(t, err)
		assert.Equal(t, "sd-3c", result.Id)
	})

	t.Run("soft-deleted documents are excluded from count and totals", func(t *testing.T) {
		repo.Coll.Drop(ctx)

		exec := testkube.TestWorkflowExecution{
			Id:   "sd-4",
			Name: "sd-name-4",
			Workflow: &testkube.TestWorkflow{
				Name: "wf-count",
				Spec: &testkube.TestWorkflowSpec{},
			},
		}
		require.NoError(t, repo.Insert(ctx, exec))

		count, err := repo.Count(ctx, testworkflow.NewExecutionsFilter().WithName("wf-count"))
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		err = repo.DeleteByTestWorkflow(ctx, "wf-count")
		require.NoError(t, err)

		count, err = repo.Count(ctx, testworkflow.NewExecutionsFilter().WithName("wf-count"))
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
