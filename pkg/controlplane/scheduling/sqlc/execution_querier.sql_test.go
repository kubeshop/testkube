package sqlc

import (
	"context"
	"testing"

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

	mock.ExpectQuery(`SELECT[\s\S]*WHERE r.status = ANY\(\$1::text\[\]\)[\s\S]*\$2::timestamptz IS NULL[\s\S]*ORDER BY e.scheduled_at, e.id[\s\S]*LIMIT \$4::int`).
		WithArgs([]string{"assigned", "starting"}, pgtype.Timestamptz{}, "", int32(100)).
		WillReturnRows(rows)

	result, err := queries.GetExecutionsByStatuses(context.Background(), GetExecutionsByStatusesParams{
		Statuses:         []string{"assigned", "starting"},
		AfterScheduledAt: pgtype.Timestamptz{},
		AfterExecutionID: "",
		RowLimit:         100,
	})

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
