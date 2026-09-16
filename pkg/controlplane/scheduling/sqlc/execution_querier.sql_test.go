package sqlc

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExecutionsByStatuses_UsesLimit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	rows := mock.NewRows([]string{"id"})

	mock.ExpectQuery(`SELECT[\s\S]*WHERE e.status = ANY\(\$1::text\[\]\)[\s\S]*COALESCE\(e.status_at, e.scheduled_at\) <= \$2::timestamptz[\s\S]*\$3::timestamptz IS NULL[\s\S]*ORDER BY COALESCE\(e.status_at, e.scheduled_at\), e.id[\s\S]*LIMIT \$5::int`).
		WithArgs([]string{"assigned", "starting"}, pgtype.Timestamptz{Time: time.Unix(1, 0), Valid: true}, pgtype.Timestamptz{}, "", int32(100)).
		WillReturnRows(rows)

	result, err := queries.GetExecutionsByStatuses(context.Background(), GetExecutionsByStatusesParams{
		Statuses:         []string{"assigned", "starting"},
		SnapshotBefore:   pgtype.Timestamptz{Time: time.Unix(1, 0), Valid: true},
		AfterPendingAt:   pgtype.Timestamptz{},
		AfterExecutionID: "",
		RowLimit:         100,
	})

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
