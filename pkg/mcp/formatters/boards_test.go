package formatters

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/mcp/boards"
)

func TestFormatBoardList(t *testing.T) {
	out, err := FormatBoardList(`[{"id":"b1","slug":"s","name":"N","shared":false,"analysisCount":2,"isUserFavorite":true,"layoutSummary":{"rows":1,"cells":2}}]`, 0, 1)
	require.NoError(t, err)
	assert.JSONEq(t, `{"boards":[{"id":"b1","slug":"s","name":"N","shared":false,"reports":2,"pinned":true}],"page":0,"hasMore":true}`, out)

	out, err = FormatBoardList(`[]`, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, "No boards found.", out)
}

func TestReportData(t *testing.T) {
	t.Run("pass-fail selects the measure", func(t *testing.T) {
		raw := `{"ratioStats":{"total":75.5,"values":[["2026-09-01",50],["2026-09-02",100]]},
			"totalStats":{"total":8,"values":[["2026-09-01",4],["2026-09-02",4]]},
			"failedStats":{"total":2,"values":[["2026-09-01",2],["2026-09-02",0]]}}`
		data, err := ReportData(boards.KindPassFail, "failed-count", raw, 0)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, "failed-count", m["measure"])
		assert.Equal(t, float64(2), m["total"])
		assert.Equal(t, float64(75.5), m["passRatio"])
		assert.Equal(t, 2, m["points"])
	})

	t.Run("pass-fail downsamples", func(t *testing.T) {
		values := make([]string, 0, 100)
		for i := 0; i < 100; i++ {
			values = append(values, fmt.Sprintf(`["d%d",%d]`, i, i))
		}
		raw := `{"ratioStats":{"total":1,"values":[` + strings.Join(values, ",") + `]},"totalStats":{"total":0,"values":[]},"failedStats":{"total":0,"values":[]}}`
		data, err := ReportData(boards.KindPassFail, "", raw, 10)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, "ratio", m["measure"])
		assert.Equal(t, 100, m["points"])
		assert.Len(t, m["samples"], 10)
	})

	t.Run("executions groups", func(t *testing.T) {
		raw := `{"count":{"total":5,"values":[["passed",4],["failed",1]]},"duration":{"total":900,"values":[["passed",100],["failed",800]]}}`
		data, err := ReportData(boards.KindExecutions, "duration", raw, 0)
		require.NoError(t, err)
		out, _ := json.Marshal(data)
		assert.JSONEq(t, `{"measure":"duration","unit":"avg ms","total":900,"groups":[{"group":"passed","value":100},{"group":"failed","value":800}]}`, string(out))
	})

	t.Run("workflows are ranked and truncated", func(t *testing.T) {
		items := make([]string, 0, 30)
		for i := 0; i < 30; i++ {
			items = append(items, fmt.Sprintf(`{"name":"wf%d","totalExecutionCount":%d,"failedExecutionCount":1,"averageDuration":10,"p95Duration":20,"lastRunAt":"2026-09-01T00:00:00Z"}`, i, i+1))
		}
		data, err := ReportData(boards.KindWorkflows, "", "["+strings.Join(items, ",")+"]", 0)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, 30, m["workflowCount"])
		assert.Equal(t, true, m["truncated"])
		workflows := m["workflows"].([]formattedWorkflowSummary)
		require.Len(t, workflows, maxRenderedWorkflows)
		assert.Equal(t, "wf29", workflows[0].Name)
		assert.InDelta(t, 96.67, workflows[0].PassRate, 0.01)
	})

	t.Run("time-series summarizes segments", func(t *testing.T) {
		raw := `[{"ts":1,"value":3,"segments":[{"label":"passed","value":2},{"label":"failed","value":1}]},{"ts":2,"value":0}]`
		data, err := ReportData(boards.KindTimeSeries, "execution-count", raw, 0)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, 2, m["pointCount"])
		assert.Len(t, m["series"], 2)
	})
}
