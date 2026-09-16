package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pagedExecution struct {
	pendingAt   time.Time
	executionID string
}

func TestExecutionBatchPagerNext_RevisitsOlderTransitionsInNextSnapshot(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	items := []pagedExecution{
		{pendingAt: base.Add(1 * time.Second), executionID: "001"},
	}

	now := base.Add(1 * time.Second)
	pager := executionBatchPager[pagedExecution]{
		now: func() time.Time { return now },
	}
	fetch := func(snapshotBefore time.Time, after *executionBatchCursor) ([]pagedExecution, error) {
		var eligible []pagedExecution
		for _, item := range items {
			if item.pendingAt.After(snapshotBefore) {
				continue
			}
			eligible = append(eligible, item)
		}

		start := 0
		if after != nil {
			start = len(eligible)
			for i, item := range eligible {
				if item.pendingAt.After(after.pendingAt) ||
					(item.pendingAt.Equal(after.pendingAt) && item.executionID > after.executionID) {
					start = i
					break
				}
			}
		}

		end := start + 1
		if end > len(eligible) {
			end = len(eligible)
		}
		if start >= len(eligible) {
			return nil, nil
		}
		return eligible[start:end], nil
	}
	cursorOf := func(item pagedExecution) *executionBatchCursor {
		return &executionBatchCursor{
			pendingAt:   item.pendingAt,
			executionID: item.executionID,
		}
	}

	page1, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[0]}, page1)

	page2, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[0]}, page2)

	items = append(items, pagedExecution{
		pendingAt:   base.Add(3 * time.Second),
		executionID: "000",
	})
	now = base.Add(3 * time.Second)

	page3, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{items[0]}, page3)

	page4, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []pagedExecution{{pendingAt: base.Add(3 * time.Second), executionID: "000"}}, page4)
}
