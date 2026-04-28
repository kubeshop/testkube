package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/test/fixtures"
)

func testUpdateReport(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("report-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	report1 := &testkube.TestWorkflowReport{
		Ref:  "step-1",
		Kind: "junit",
		File: "report-1.xml",
		Summary: &testkube.TestWorkflowReportSummary{
			Tests:    10,
			Passed:   8,
			Failed:   1,
			Skipped:  1,
			Duration: 5000,
		},
	}
	err = repo.UpdateReport(ctx, execution.Id, report1)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	assert.Equal(t, "step-1", got.Reports[0].Ref)
	assert.Equal(t, int32(10), got.Reports[0].Summary.Tests)

	report2 := &testkube.TestWorkflowReport{
		Ref:  "step-2",
		Kind: "junit",
		File: "report-2.xml",
		Summary: &testkube.TestWorkflowReportSummary{
			Tests:    5,
			Passed:   5,
			Duration: 2000,
		},
	}
	err = repo.UpdateReport(ctx, execution.Id, report2)
	require.NoError(t, err)

	got, err = repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	require.Len(t, got.Reports, 2)

	assert.Equal(t, "step-1", got.Reports[0].Ref, "first report should be step-1 (inserted first)")
	assert.Equal(t, "step-2", got.Reports[1].Ref, "second report should be step-2 (inserted second)")
}

func testUpdateOutput(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("output-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	outputs := []testkube.TestWorkflowOutput{
		{Ref: "step-1", Name: "artifact", Value: map[string]interface{}{"path": "/results/output.log"}},
		{Ref: "step-2", Name: "result", Value: map[string]interface{}{"status": "ok"}},
	}
	err = repo.UpdateOutput(ctx, execution.Id, outputs)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	require.Len(t, got.Output, 2)
	assert.Equal(t, "step-1", got.Output[0].Ref)
	assert.Equal(t, "step-2", got.Output[1].Ref)
}

func testUpdateResourceAggregations(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("resource-agg-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	agg := &testkube.TestWorkflowExecutionResourceAggregationsReport{
		Global: map[string]map[string]testkube.TestWorkflowExecutionResourceAggregations{
			"cpu": {
				"usage": {Total: 1.5, Avg: 0.75, Min: 0.5, Max: 1.0},
			},
		},
	}
	err = repo.UpdateResourceAggregations(ctx, execution.Id, agg)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	require.NotNil(t, got.ResourceAggregations)
	assert.Equal(t, 1.5, got.ResourceAggregations.Global["cpu"]["usage"].Total)
}
