package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pagedExecution struct {
	scheduledAt time.Time
	executionID string
}

func TestExecutionBatchPagerNext_AdvancesAndWraps(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	items := []pagedExecution{
		{scheduledAt: base.Add(1 * time.Second), executionID: "001"},
		{scheduledAt: base.Add(2 * time.Second), executionID: "002"},
		{scheduledAt: base.Add(3 * time.Second), executionID: "003"},
		{scheduledAt: base.Add(4 * time.Second), executionID: "004"},
		{scheduledAt: base.Add(5 * time.Second), executionID: "005"},
	}

	var pager executionBatchPager[pagedExecution]
	fetch := func(after *executionBatchCursor) ([]pagedExecution, error) {
		start := 0
		if after != nil {
			start = len(items)
			for i, item := range items {
				if item.scheduledAt.After(after.scheduledAt) ||
					(item.scheduledAt.Equal(after.scheduledAt) && item.executionID > after.executionID) {
					start = i
					break
				}
			}
		}

		end := start + 2
		if end > len(items) {
			end = len(items)
		}
		if start >= len(items) {
			return nil, nil
		}
		return items[start:end], nil
	}
	cursorOf := func(item pagedExecution) *executionBatchCursor {
		return &executionBatchCursor{
			scheduledAt: item.scheduledAt,
			executionID: item.executionID,
		}
	}

	page1, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[0], items[1]}, page1)

	page2, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[2], items[3]}, page2)

	page3, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[4]}, page3)

	page4, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[0], items[1]}, page4)
}
