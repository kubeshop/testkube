package datefilter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsPassingDateFilterWhenInvalidStartAndEndDateThenValidateAll(t *testing.T) {
	t.Parallel()

	assertion := require.New(t)
	dFilter := NewDateFilter("", "")
	assertion.True(dFilter.IsPassing(time.Now()), "Date filter should pass any value if the start date and end date are empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(5, 5, 20)), "Date filter should pass any value if the start date and end date are empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(-5, -5, -5)), "Date filter should pass any value if the start date and end date are empty")
}

func TestIsPassingDateFilterWhenStartIsValidAndEndDateInvalidThenValidateAllAfterStart(t *testing.T) {
	t.Parallel()

	assertion := require.New(t)
	fiveDaysAgo := time.Now().AddDate(0, 0, -5).Format(DateFormatISO8601)
	dFilter := NewDateFilter(fiveDaysAgo, "")
	assertion.True(dFilter.IsPassing(time.Now()), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(0, 0, -2)), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(2, 2, 2)), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.False(dFilter.IsPassing(time.Now().AddDate(-2, 2, 2)), "Date filter should fail any value before start date even if the end date is invalid or empty")
}

func TestIsPassingDateFilterWhenStartIsValidAndEndValidThenOnlyDatesBetweenAreValidated(t *testing.T) {
	t.Parallel()

	assertion := require.New(t)
	tenDaysAgo := time.Now().AddDate(0, 0, -10).Format(DateFormatISO8601)
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format(DateFormatISO8601)
	dFilter := NewDateFilter(tenDaysAgo, twoDaysAgo)
	assertion.True(dFilter.IsPassing(time.Now().AddDate(0, 0, -3)), "Date filter should pass any value after start date and before end date")
	assertion.False(dFilter.IsPassing(time.Now().AddDate(0, 0, -15)), "Date filter should fail any value before start date")
	assertion.False(dFilter.IsPassing(time.Now()), "Date filter should fail any value after start date")
}

func TestIsPassingDateFilterWhenStartAndEndDateAreTheSameThenValidateForSameDate(t *testing.T) {
	t.Parallel()

	assertion := require.New(t)
	now := time.Now().Format(DateFormatISO8601)
	dFilter := NewDateFilter(now, now)
	assertion.False(dFilter.IsPassing(time.Now().AddDate(5, 5, 20)), "Date filter should not pass any value if it is not between the start and end")
}
