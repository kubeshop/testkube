package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// Mock implementations
type MockTestWorkflowExecutionQueriesInterface struct {
	mock.Mock
}

func (m *MockTestWorkflowExecutionQueriesInterface) WithTx(tx pgx.Tx) sqlc.TestWorkflowExecutionQueriesInterface {
	args := m.Called(tx)
	return args.Get(0).(sqlc.TestWorkflowExecutionQueriesInterface)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecution(ctx context.Context, id string) (sqlc.GetTestWorkflowExecutionRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(sqlc.GetTestWorkflowExecutionRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecutionByNameAndTestWorkflow(ctx context.Context, arg sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams) (sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetLatestTestWorkflowExecutionByTestWorkflow(ctx context.Context, workflowName string) (sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow, error) {
	args := m.Called(ctx, workflowName)
	return args.Get(0).(sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetLatestTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []string) ([]sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow, error) {
	args := m.Called(ctx, workflowNames)
	return args.Get(0).([]sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetRunningTestWorkflowExecutions(ctx context.Context) ([]sqlc.GetRunningTestWorkflowExecutionsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]sqlc.GetRunningTestWorkflowExecutionsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetFinishedTestWorkflowExecutions(ctx context.Context, arg sqlc.GetFinishedTestWorkflowExecutionsParams) ([]sqlc.GetFinishedTestWorkflowExecutionsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.GetFinishedTestWorkflowExecutionsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecutionsTotals(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsTotalsParams) ([]sqlc.GetTestWorkflowExecutionsTotalsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.GetTestWorkflowExecutionsTotalsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecutions(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsParams) ([]sqlc.GetTestWorkflowExecutionsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.GetTestWorkflowExecutionsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecutionsSummary(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsSummaryParams) ([]sqlc.GetTestWorkflowExecutionsSummaryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.GetTestWorkflowExecutionsSummaryRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowExecution(ctx context.Context, arg sqlc.InsertTestWorkflowExecutionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowSignature(ctx context.Context, arg sqlc.InsertTestWorkflowSignatureParams) (int32, error) {
	args := m.Called(ctx, arg)
	return int32(args.Int(0)), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowResult(ctx context.Context, arg sqlc.InsertTestWorkflowResultParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowOutput(ctx context.Context, arg sqlc.InsertTestWorkflowOutputParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowReport(ctx context.Context, arg sqlc.InsertTestWorkflowReportParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflowResourceAggregations(ctx context.Context, arg sqlc.InsertTestWorkflowResourceAggregationsParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InsertTestWorkflow(ctx context.Context, arg sqlc.InsertTestWorkflowParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) UpdateTestWorkflowExecution(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) UpdateTestWorkflowExecutionResult(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionResultParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) UpdateExecutionStatusAt(ctx context.Context, arg sqlc.UpdateExecutionStatusAtParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) UpdateTestWorkflowExecutionReport(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionReportParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) UpdateTestWorkflowExecutionResourceAggregations(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionResourceAggregationsParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowOutputs(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowExecutionsByTestWorkflow(ctx context.Context, workflowName string) error {
	args := m.Called(ctx, workflowName)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteAllTestWorkflowExecutions(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []string) error {
	args := m.Called(ctx, workflowNames)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowMetrics(ctx context.Context, arg sqlc.GetTestWorkflowMetricsParams) ([]sqlc.GetTestWorkflowMetricsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.GetTestWorkflowMetricsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetPreviousFinishedState(ctx context.Context, arg sqlc.GetPreviousFinishedStateParams) (pgtype.Text, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgtype.Text), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetTestWorkflowExecutionTags(ctx context.Context, workflowName string) ([]sqlc.GetTestWorkflowExecutionTagsRow, error) {
	args := m.Called(ctx, workflowName)
	return args.Get(0).([]sqlc.GetTestWorkflowExecutionTagsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) InitTestWorkflowExecution(ctx context.Context, arg sqlc.InitTestWorkflowExecutionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) AssignTestWorkflowExecution(ctx context.Context, arg sqlc.AssignTestWorkflowExecutionParams) (string, error) {
	args := m.Called(ctx, arg)
	return args.String(0), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) GetUnassignedTestWorkflowExecutions(ctx context.Context) ([]sqlc.GetUnassignedTestWorkflowExecutionsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]sqlc.GetUnassignedTestWorkflowExecutionsRow), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) AbortTestWorkflowExecutionIfQueued(ctx context.Context, arg sqlc.AbortTestWorkflowExecutionIfQueuedParams) (string, error) {
	args := m.Called(ctx, arg)
	return args.String(0), args.Error(1)
}

func (m *MockTestWorkflowExecutionQueriesInterface) AbortTestWorkflowResultIfQueued(ctx context.Context, arg sqlc.AbortTestWorkflowResultIfQueuedParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowSignatures(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowResult(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowReports(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflowResourceAggregations(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockTestWorkflowExecutionQueriesInterface) DeleteTestWorkflow(ctx context.Context, arg sqlc.DeleteTestWorkflowParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// Mock DatabaseInterface
type MockDatabaseInterface struct {
	mock.Mock
}

func (m *MockDatabaseInterface) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

// Mock Tx
type MockTx struct {
	mock.Mock
}

// Begin mocks the Begin method
func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

// Commit mocks the Commit method
func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Rollback mocks the Rollback method
func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// CopyFrom mocks the CopyFrom method
func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	args := m.Called(ctx, tableName, columnNames, rowSrc)
	return args.Get(0).(int64), args.Error(1)
}

// SendBatch mocks the SendBatch method
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	args := m.Called(ctx, b)
	return args.Get(0).(pgx.BatchResults)
}

// LargeObjects mocks the LargeObjects method
func (m *MockTx) LargeObjects() pgx.LargeObjects {
	args := m.Called()
	return args.Get(0).(pgx.LargeObjects)
}

// Prepare mocks the Prepare method
func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	args := m.Called(ctx, name, sql)
	return args.Get(0).(*pgconn.StatementDescription), args.Error(1)
}

// Exec mocks the Exec method
func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

// Query mocks the Query method
func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgx.Rows), callArgs.Error(1)
}

// QueryRow mocks the QueryRow method
func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgx.Row)
}

// Conn mocks the Conn method
func (m *MockTx) Conn() *pgx.Conn {
	args := m.Called()
	return args.Get(0).(*pgx.Conn)
}

// Mock SequenceRepository
type MockSequenceRepository struct {
	mock.Mock
}

func (m *MockSequenceRepository) GetNextExecutionNumber(ctx context.Context, name string, executionType sequence.ExecutionType) (int32, error) {
	args := m.Called(ctx, name, executionType)
	return int32(args.Int(0)), args.Error(1)
}

func (m *MockSequenceRepository) DeleteExecutionNumber(ctx context.Context, name string, executionType sequence.ExecutionType) error {
	args := m.Called(ctx, name, executionType)
	return args.Error(0)
}

func (m *MockSequenceRepository) DeleteAllExecutionNumbers(ctx context.Context, executionType sequence.ExecutionType) error {
	args := m.Called(ctx, executionType)
	return args.Error(0)
}

func (m *MockSequenceRepository) DeleteExecutionNumbers(ctx context.Context, names []string, executionType sequence.ExecutionType) error {
	args := m.Called(ctx, names, executionType)
	return args.Error(0)
}

// Mock Filter
type MockFilter struct {
	mock.Mock
}

func (m *MockFilter) Name() string {
	return m.Called().String(0)
}

func (m *MockFilter) NameDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Names() []string {
	return m.Called().Get(0).([]string)
}

func (m *MockFilter) NamesDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) LastNDays() int {
	return m.Called().Int(0)
}

func (m *MockFilter) LastNDaysDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) StartDate() time.Time {
	return m.Called().Get(0).(time.Time)
}

func (m *MockFilter) StartDateDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) EndDate() time.Time {
	return m.Called().Get(0).(time.Time)
}

func (m *MockFilter) EndDateDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Statuses() []testkube.TestWorkflowStatus {
	return m.Called().Get(0).([]testkube.TestWorkflowStatus)
}

func (m *MockFilter) StatusesDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Page() int {
	return m.Called().Int(0)
}

func (m *MockFilter) PageSize() int {
	return m.Called().Int(0)
}

func (m *MockFilter) TextSearch() string {
	return m.Called().String(0)
}

func (m *MockFilter) TextSearchDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Selector() string {
	return m.Called().String(0)
}

func (m *MockFilter) TagSelector() string {
	return m.Called().String(0)
}

func (m *MockFilter) LabelSelector() *testworkflow.LabelSelector {
	return m.Called().Get(0).(*testworkflow.LabelSelector)
}

func (m *MockFilter) ActorName() string {
	return m.Called().String(0)
}

func (m *MockFilter) ActorNameDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) ActorType() testkube.TestWorkflowRunningContextActorType {
	return m.Called().Get(0).(testkube.TestWorkflowRunningContextActorType)
}

func (m *MockFilter) ActorTypeDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) GroupID() string {
	return m.Called().String(0)
}

func (m *MockFilter) GroupIDDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) RunnerID() string {
	return m.Called().String(0)
}

func (m *MockFilter) RunnerIDDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Initialized() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) InitializedDefined() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) Assigned() bool {
	return m.Called().Bool(0)
}

func (m *MockFilter) AssignedDefined() bool {
	return m.Called().Bool(0)
}

// Helper functions for tests
func createTestExecution() *testkube.TestWorkflowExecution {
	status := testkube.PASSED_TestWorkflowStatus
	return &testkube.TestWorkflowExecution{
		Id:          "test-id",
		Name:        "test-execution",
		GroupId:     "group-1",
		RunnerId:    "runner-1",
		Namespace:   "default",
		Number:      1,
		ScheduledAt: time.Now(),
		StatusAt:    time.Now(),
		Result: &testkube.TestWorkflowResult{
			Status: &status,
		},
		Workflow: &testkube.TestWorkflow{
			Name:      "test-workflow",
			Namespace: "default",
			Spec:      &testkube.TestWorkflowSpec{},
		},
		Tags: map[string]string{
			"env": "test",
		},
	}
}

func createTestFilter() *MockFilter {
	filter := &MockFilter{}
	filter.On("Page").Return(0)
	filter.On("PageSize").Return(100)
	filter.On("NameDefined").Return(false)
	filter.On("NamesDefined").Return(false)
	filter.On("TextSearchDefined").Return(false)
	filter.On("StartDateDefined").Return(false)
	filter.On("EndDateDefined").Return(false)
	filter.On("LastNDaysDefined").Return(false)
	filter.On("StatusesDefined").Return(false)
	filter.On("RunnerIDDefined").Return(false)
	filter.On("AssignedDefined").Return(false)
	filter.On("ActorNameDefined").Return(false)
	filter.On("ActorTypeDefined").Return(false)
	filter.On("GroupIDDefined").Return(false)
	filter.On("InitializedDefined").Return(false)
	filter.On("Selector").Return("")
	filter.On("TagSelector").Return("")
	filter.On("LabelSelector").Return((*testworkflow.LabelSelector)(nil))
	return filter
}

func createTestRow() sqlc.GetTestWorkflowExecutionRow {
	tagsJSON, _ := json.Marshal(map[string]string{"env": "test"})
	return sqlc.GetTestWorkflowExecutionRow{
		ID:                  "test-id",
		GroupID:             pgtype.Text{String: "group-1", Valid: true},
		RunnerID:            pgtype.Text{String: "runner-1", Valid: true},
		Name:                "test-execution",
		Namespace:           pgtype.Text{String: "default", Valid: true},
		Number:              pgtype.Int4{Int32: 1, Valid: true},
		ScheduledAt:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
		StatusAt:            pgtype.Timestamptz{Time: time.Now(), Valid: true},
		DisableWebhooks:     pgtype.Bool{Bool: false, Valid: true},
		Tags:                tagsJSON,
		RunningContext:      []byte(`{}`),
		ConfigParams:        []byte(`{}`),
		Status:              pgtype.Text{String: "passed", Valid: true},
		PredictedStatus:     pgtype.Text{String: "passed", Valid: true},
		QueuedAt:            pgtype.Timestamptz{Time: time.Now(), Valid: true},
		StartedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
		FinishedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Duration:            pgtype.Text{String: "5m", Valid: true},
		TotalDuration:       pgtype.Text{String: "5m", Valid: true},
		DurationMs:          pgtype.Int4{Int32: 300000, Valid: true},
		PausedMs:            pgtype.Int4{Int32: 0, Valid: true},
		TotalDurationMs:     pgtype.Int4{Int32: 300000, Valid: true},
		Pauses:              []byte(`[]`),
		Initialization:      []byte(`{}`),
		Steps:               []byte(`{}`),
		WorkflowName:        pgtype.Text{String: "test-workflow", Valid: true},
		WorkflowNamespace:   pgtype.Text{String: "default", Valid: true},
		WorkflowDescription: pgtype.Text{String: "Test workflow", Valid: true},
		WorkflowLabels:      []byte(`{}`),
		WorkflowAnnotations: []byte(`{}`),
		WorkflowCreated:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		WorkflowUpdated:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		WorkflowSpec:        []byte(`{}`),
		WorkflowReadOnly:    pgtype.Bool{Bool: false, Valid: true},
		WorkflowStatus:      []byte(`{}`),
		SignaturesJson:      []byte(`[]`),
		OutputsJson:         []byte(`[]`),
		ReportsJson:         []byte(`[]`),
	}
}

// Unit Tests

func TestPostgresRepository_Get(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	mockDB := &MockDatabaseInterface{}

	repo := &PostgresRepository{
		db:      mockDB,
		queries: mockQueries,
	}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		executionID := "test-id"

		row := createTestRow()
		mockQueries.On("GetTestWorkflowExecution", ctx, executionID).Return(row, nil)

		result, err := repo.Get(ctx, executionID)

		assert.NoError(t, err)
		assert.Equal(t, executionID, result.Id)
		assert.Equal(t, "test-execution", result.Name)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		executionID := "not-found"

		mockQueries.On("GetTestWorkflowExecution", ctx, executionID).Return(sqlc.GetTestWorkflowExecutionRow{}, pgx.ErrNoRows)

		_, err := repo.Get(ctx, executionID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), pgx.ErrNoRows.Error())
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetByNameAndTestWorkflow(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		name := "test-execution"
		workflowName := "test-workflow"

		row := sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow(createTestRow())
		params := sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams{
			Name:         name,
			WorkflowName: workflowName,
		}

		mockQueries.On("GetTestWorkflowExecutionByNameAndTestWorkflow", ctx, params).Return(row, nil)

		result, err := repo.GetByNameAndTestWorkflow(ctx, name, workflowName)

		assert.NoError(t, err)
		assert.Equal(t, "test-id", result.Id)
		assert.Equal(t, name, result.Name)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		name := "not-found"
		workflowName := "test-workflow"

		params := sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams{
			Name:         name,
			WorkflowName: workflowName,
		}

		mockQueries.On("GetTestWorkflowExecutionByNameAndTestWorkflow", ctx, params).Return(sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow{}, pgx.ErrNoRows)

		_, err := repo.GetByNameAndTestWorkflow(ctx, name, workflowName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), pgx.ErrNoRows.Error())
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetLatestByTestWorkflow(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		workflowName := "test-workflow"

		row := sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow(createTestRow())
		mockQueries.On("GetLatestTestWorkflowExecutionByTestWorkflow", ctx, workflowName).Return(row, nil)

		result, err := repo.GetLatestByTestWorkflow(ctx, workflowName)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-id", result.Id)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		workflowName := "not-found"

		mockQueries.On("GetLatestTestWorkflowExecutionByTestWorkflow", ctx, workflowName).Return(sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow{}, pgx.ErrNoRows)

		result, err := repo.GetLatestByTestWorkflow(ctx, workflowName)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetLatestByTestWorkflows(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		workflowNames := []string{"workflow1", "workflow2"}

		rows := []sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow{
			sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow(createTestRow()),
		}

		mockQueries.On("GetLatestTestWorkflowExecutionsByTestWorkflows", ctx, workflowNames).Return(rows, nil)

		result, err := repo.GetLatestByTestWorkflows(ctx, workflowNames)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockQueries.AssertExpectations(t)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		ctx := context.Background()
		workflowNames := []string{}

		result, err := repo.GetLatestByTestWorkflows(ctx, workflowNames)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestPostgresRepository_GetRunning(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		repo := &PostgresRepository{queries: mockQueries}
		ctx := context.Background()

		row := createTestRow()
		row.Status = pgtype.Text{String: "running", Valid: true}
		rows := []sqlc.GetRunningTestWorkflowExecutionsRow{
			sqlc.GetRunningTestWorkflowExecutionsRow(row),
		}

		mockQueries.On("GetRunningTestWorkflowExecutions", ctx).Return(rows, nil)

		result, err := repo.GetRunning(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "test-id", result[0].Id)
		mockQueries.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		repo := &PostgresRepository{queries: mockQueries}
		ctx := context.Background()

		mockQueries.On("GetRunningTestWorkflowExecutions", ctx).Return([]sqlc.GetRunningTestWorkflowExecutionsRow{}, errors.New("database error"))

		result, err := repo.GetRunning(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetExecutionsTotals(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		filter := createTestFilter()

		rows := []sqlc.GetTestWorkflowExecutionsTotalsRow{
			{
				Status: pgtype.Text{String: string(testkube.PASSED_TestWorkflowStatus), Valid: true},
				Count:  5,
			},
			{
				Status: pgtype.Text{String: string(testkube.FAILED_TestWorkflowStatus), Valid: true},
				Count:  3,
			},
		}

		mockQueries.On("GetTestWorkflowExecutionsTotals", ctx, mock.AnythingOfType("sqlc.GetTestWorkflowExecutionsTotalsParams")).Return(rows, nil)

		result, err := repo.GetExecutionsTotals(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, int32(5), result.Passed)
		assert.Equal(t, int32(3), result.Failed)
		assert.Equal(t, int32(8), result.Results)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetExecutions(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		filter := createTestFilter()

		rows := []sqlc.GetTestWorkflowExecutionsRow{
			sqlc.GetTestWorkflowExecutionsRow(createTestRow()),
		}

		mockQueries.On("GetTestWorkflowExecutions", ctx, mock.AnythingOfType("sqlc.GetTestWorkflowExecutionsParams")).Return(rows, nil)

		result, err := repo.GetExecutions(ctx, filter)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_Insert(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockDB := &MockDatabaseInterface{}
		mockTx := &MockTx{}
		repo := &PostgresRepository{
			db:      mockDB,
			queries: mockQueries,
		}

		ctx := context.Background()
		execution := createTestExecution()

		// Mock transaction
		mockDB.On("Begin", ctx).Return(mockTx, nil)
		mockTx.On("Rollback", ctx).Return(nil)
		mockTx.On("Commit", ctx).Return(nil)
		mockQueries.On("WithTx", mockTx).Return(mockQueries)

		// Mock insert operations
		mockQueries.On("InsertTestWorkflowExecution", ctx, mock.AnythingOfType("sqlc.InsertTestWorkflowExecutionParams")).Return(nil)
		mockQueries.On("InsertTestWorkflowResult", ctx, mock.AnythingOfType("sqlc.InsertTestWorkflowResultParams")).Return(nil)
		mockQueries.On("InsertTestWorkflow", ctx, mock.AnythingOfType("sqlc.InsertTestWorkflowParams")).Return(nil)

		err := repo.Insert(ctx, *execution)

		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("TransactionError", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockDB := &MockDatabaseInterface{}
		mockTx := &MockTx{}
		repo := &PostgresRepository{
			db:      mockDB,
			queries: mockQueries,
		}

		ctx := context.Background()
		execution := createTestExecution()

		mockDB.On("Begin", ctx).Return(mockTx, errors.New("transaction error"))

		err := repo.Insert(ctx, *execution)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")
		mockDB.AssertExpectations(t)
	})
}

func TestPostgresRepository_UpdateResult(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		repo := &PostgresRepository{queries: mockQueries}
		ctx := context.Background()
		id := "test-id"
		result := &testkube.TestWorkflowResult{
			Status:     &[]testkube.TestWorkflowStatus{testkube.PASSED_TestWorkflowStatus}[0],
			FinishedAt: time.Now(),
		}

		mockQueries.On("UpdateTestWorkflowExecutionResult", ctx, mock.AnythingOfType("sqlc.UpdateTestWorkflowExecutionResultParams")).Return(nil)
		mockQueries.On("UpdateExecutionStatusAt", ctx, mock.AnythingOfType("sqlc.UpdateExecutionStatusAtParams")).Return(nil)

		err := repo.UpdateResult(ctx, id, result)

		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
	})

	t.Run("UpdateError", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		repo := &PostgresRepository{queries: mockQueries}
		ctx := context.Background()
		id := "test-id"
		result := &testkube.TestWorkflowResult{
			Status: &[]testkube.TestWorkflowStatus{testkube.PASSED_TestWorkflowStatus}[0],
		}

		mockQueries.On("UpdateTestWorkflowExecutionResult", ctx, mock.AnythingOfType("sqlc.UpdateTestWorkflowExecutionResultParams")).Return(errors.New("update error"))

		err := repo.UpdateResult(ctx, id, result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update error")
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_DeleteByTestWorkflow(t *testing.T) {
	t.Run("Success", func(t *testing.T) {

		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockSeq := &MockSequenceRepository{}
		repo := &PostgresRepository{
			queries:            mockQueries,
			sequenceRepository: mockSeq,
		}

		ctx := context.Background()
		workflowName := "test-workflow"

		mockSeq.On("DeleteExecutionNumber", ctx, workflowName, sequence.ExecutionTypeTestWorkflow).Return(nil)
		mockQueries.On("DeleteTestWorkflowExecutionsByTestWorkflow", ctx, workflowName).Return(nil)

		err := repo.DeleteByTestWorkflow(ctx, workflowName)

		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
		mockSeq.AssertExpectations(t)
	})

	t.Run("SequenceError", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockSeq := &MockSequenceRepository{}
		repo := &PostgresRepository{
			queries:            mockQueries,
			sequenceRepository: mockSeq,
		}

		ctx := context.Background()
		workflowName := "test-workflow"

		mockSeq.On("DeleteExecutionNumber", ctx, workflowName, sequence.ExecutionTypeTestWorkflow).Return(errors.New("sequence error"))

		err := repo.DeleteByTestWorkflow(ctx, workflowName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sequence error")
		mockSeq.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetTestWorkflowMetrics(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		name := "test-workflow"
		limit := 10
		last := 7

		rows := []sqlc.GetTestWorkflowMetricsRow{
			{
				ExecutionID: "exec1",
				GroupID:     pgtype.Text{String: "group1", Valid: true},
				Duration:    pgtype.Text{String: "5m", Valid: true},
				DurationMs:  pgtype.Int4{Int32: 300000, Valid: true},
				Status:      pgtype.Text{String: string(testkube.PASSED_TestWorkflowStatus), Valid: true},
				Name:        "execution1",
				StartTime:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
				RunnerID:    pgtype.Text{String: "runner1", Valid: true},
			},
		}

		params := sqlc.GetTestWorkflowMetricsParams{
			WorkflowName: name,
			LastNDays:    int32(last),
			Lmt:          int32(limit),
		}

		mockQueries.On("GetTestWorkflowMetrics", ctx, params).Return(rows, nil)

		result, err := repo.GetTestWorkflowMetrics(ctx, name, limit, last)

		assert.NoError(t, err)
		assert.Len(t, result.Executions, 1)
		assert.Equal(t, "exec1", result.Executions[0].ExecutionId)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_GetNextExecutionNumber(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSeq := &MockSequenceRepository{}
		repo := &PostgresRepository{
			sequenceRepository: mockSeq,
		}

		ctx := context.Background()
		name := "test-workflow"
		expectedNumber := 5

		mockSeq.On("GetNextExecutionNumber", ctx, name, sequence.ExecutionTypeTestWorkflow).Return(expectedNumber, nil)

		result, err := repo.GetNextExecutionNumber(ctx, name)

		assert.NoError(t, err)
		assert.Equal(t, int32(expectedNumber), result)
		mockSeq.AssertExpectations(t)
	})

	t.Run("NoSequenceRepository", func(t *testing.T) {
		repo := &PostgresRepository{
			sequenceRepository: nil,
		}
		ctx := context.Background()
		name := "test-workflow"

		result, err := repo.GetNextExecutionNumber(ctx, name)

		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Contains(t, err.Error(), "no sequence repository provided")
	})
}

func TestPostgresRepository_Assign(t *testing.T) {
	mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
	repo := &PostgresRepository{queries: mockQueries}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		id := "test-id"
		prevRunnerID := "old-runner"
		newRunnerID := "new-runner"
		assignedAt := time.Now()

		params := sqlc.AssignTestWorkflowExecutionParams{
			ID:           id,
			PrevRunnerID: prevRunnerID,
			NewRunnerID:  newRunnerID,
			AssignedAt:   toPgTimestamp(assignedAt),
		}

		mockQueries.On("AssignTestWorkflowExecution", ctx, params).Return(id, nil)

		result, err := repo.Assign(ctx, id, prevRunnerID, newRunnerID, &assignedAt)

		assert.NoError(t, err)
		assert.True(t, result)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		id := "test-id"
		prevRunnerID := "old-runner"
		newRunnerID := "new-runner"
		assignedAt := time.Now()

		params := sqlc.AssignTestWorkflowExecutionParams{
			ID:           id,
			PrevRunnerID: prevRunnerID,
			NewRunnerID:  newRunnerID,
			AssignedAt:   toPgTimestamp(assignedAt),
		}

		mockQueries.On("AssignTestWorkflowExecution", ctx, params).Return("", pgx.ErrNoRows)

		result, err := repo.Assign(ctx, id, prevRunnerID, newRunnerID, &assignedAt)

		assert.NoError(t, err)
		assert.False(t, result)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_AbortIfQueued(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockDB := &MockDatabaseInterface{}
		mockTx := &MockTx{}
		repo := &PostgresRepository{
			db:      mockDB,
			queries: mockQueries,
		}

		ctx := context.Background()
		id := "test-id"

		// Mock transaction
		mockDB.On("Begin", ctx).Return(mockTx, nil)
		mockTx.On("Rollback", ctx).Return(nil)
		mockTx.On("Commit", ctx).Return(nil)
		mockQueries.On("WithTx", mockTx).Return(mockQueries)

		// Mock abort operations
		mockQueries.On("AbortTestWorkflowExecutionIfQueued", ctx, mock.AnythingOfType("sqlc.AbortTestWorkflowExecutionIfQueuedParams")).Return(id, nil)
		mockQueries.On("AbortTestWorkflowResultIfQueued", ctx, mock.AnythingOfType("sqlc.AbortTestWorkflowResultIfQueuedParams")).Return(nil)

		result, err := repo.AbortIfQueued(ctx, id)

		assert.NoError(t, err)
		assert.True(t, result)
		mockQueries.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockQueries := &MockTestWorkflowExecutionQueriesInterface{}
		mockDB := &MockDatabaseInterface{}
		mockTx := &MockTx{}
		repo := &PostgresRepository{
			db:      mockDB,
			queries: mockQueries,
		}

		ctx := context.Background()
		id := "test-id"

		// Mock transaction
		mockDB.On("Begin", ctx).Return(mockTx, nil)
		mockTx.On("Rollback", ctx).Return(nil)
		mockQueries.On("WithTx", mockTx).Return(mockQueries)

		// Mock abort operations - execution not found
		mockQueries.On("AbortTestWorkflowExecutionIfQueued", ctx, mock.AnythingOfType("sqlc.AbortTestWorkflowExecutionIfQueuedParams")).Return("", pgx.ErrNoRows)

		result, err := repo.AbortIfQueued(ctx, id)

		assert.NoError(t, err)
		assert.False(t, result)
		mockQueries.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})
}

// Test helper functions
func TestTypeConversionHelpers(t *testing.T) {
	t.Run("toPgText", func(t *testing.T) {
		// Test with non-empty string
		result := toPgText("test")
		assert.True(t, result.Valid)
		assert.Equal(t, "test", result.String)

		// Test with empty string
		result = toPgText("")
		assert.False(t, result.Valid)
	})

	t.Run("fromPgText", func(t *testing.T) {
		// Test with valid pgtype.Text
		pgText := pgtype.Text{String: "test", Valid: true}
		result := fromPgText(pgText)
		assert.Equal(t, "test", result)

		// Test with invalid pgtype.Text
		pgText = pgtype.Text{Valid: false}
		result = fromPgText(pgText)
		assert.Equal(t, "", result)
	})

	t.Run("toPgBool", func(t *testing.T) {
		result := toPgBool(true)
		assert.True(t, result.Valid)
		assert.True(t, result.Bool)

		result = toPgBool(false)
		assert.True(t, result.Valid)
		assert.False(t, result.Bool)
	})

	t.Run("fromPgBool", func(t *testing.T) {
		// Test with valid pgtype.Bool
		pgBool := pgtype.Bool{Bool: true, Valid: true}
		result := fromPgBool(pgBool)
		assert.True(t, result)

		// Test with invalid pgtype.Bool
		pgBool = pgtype.Bool{Valid: false}
		result = fromPgBool(pgBool)
		assert.False(t, result)
	})
}

func TestBuildTestWorkflowExecutionParams(t *testing.T) {
	repo := &PostgresRepository{}
	filter := testworkflow.NewExecutionsFilter().WithName("test-workflow")

	params, err := repo.buildTestWorkflowExecutionParams(filter)

	assert.NoError(t, err)
	assert.Equal(t, "test-workflow", params.WorkflowName)
}

func TestPopulateConfigParams(t *testing.T) {
	resolvedWorkflow := &testkube.TestWorkflow{
		Spec: &testkube.TestWorkflowSpec{
			Config: map[string]testkube.TestWorkflowParameterSchema{
				"param1": {
					Sensitive: true,
				},
				"param2": {
					Default_: &testkube.BoxedString{
						Value: "default-value",
					},
				},
			},
		},
	}

	configParams := map[string]testkube.TestWorkflowExecutionConfigValue{
		"param2": {
			Value: "custom-value",
		},
	}

	result := populateConfigParams(resolvedWorkflow, configParams)

	assert.Len(t, result, 2)
	assert.True(t, result["param1"].Sensitive)
	assert.True(t, result["param1"].EmptyValue)
	assert.Equal(t, "custom-value", result["param2"].Value)
	assert.Equal(t, "default-value", result["param2"].DefaultValue)
}
