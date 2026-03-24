package datefilter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsPassingDateFilterWhenInvalidStartAndEndDateThenValidateAll(t *testing.T) {
	assertion := require.New(t)
	dFilter := NewDateFilter("", "")
	assertion.True(dFilter.IsPassing(time.Now()), "Date filter should pass any value if the start date and end date are empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(5, 5, 20)), "Date filter should pass any value if the start date and end date are empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(-5, -5, -5)), "Date filter should pass any value if the start date and end date are empty")
}

func TestIsPassingDateFilterWhenStartIsValidAndEndDateInvalidThenValidateAllAfterStart(t *testing.T) {
	assertion := require.New(t)
	fiveDaysAgo := time.Now().AddDate(0, 0, -5).Format(DateFormatISO8601)
	dFilter := NewDateFilter(fiveDaysAgo, "")
	assertion.True(dFilter.IsPassing(time.Now()), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(0, 0, -2)), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.True(dFilter.IsPassing(time.Now().AddDate(2, 2, 2)), "Date filter should pass any value after start date even if the end date is invalid or empty")
	assertion.False(dFilter.IsPassing(time.Now().AddDate(-2, 2, 2)), "Date filter should fail any value before start date even if the end date is invalid or empty")
}

func TestIsPassingDateFilterWhenStartIsValidAndEndValidThenOnlyDatesBetweenAreValidated(t *testing.T) {
	assertion := require.New(t)
	tenDaysAgo := time.Now().AddDate(0, 0, -10).Format(DateFormatISO8601)
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format(DateFormatISO8601)
	dFilter := NewDateFilter(tenDaysAgo, twoDaysAgo)
	assertion.True(dFilter.IsPassing(time.Now().AddDate(0, 0, -3)), "Date filter should pass any value after start date and before end date")
	assertion.False(dFilter.IsPassing(time.Now().AddDate(0, 0, -15)), "Date filter should fail any value before start date")
	assertion.False(dFilter.IsPassing(time.Now()), "Date filter should fail any value after start date")
}

func TestIsPassingDateFilterWhenStartAndEndDateAreTheSameThenValidateForSameDate(t *testing.T) {
	assertion := require.New(t)
	nowTime := time.Now().UTC()
	now := nowTime.Format(DateFormatISO8601)
	dFilter := NewDateFilter(now, now)
	assertion.True(dFilter.IsPassing(nowTime), "Date filter should pass the current time when start and end are today")
	assertion.False(dFilter.IsPassing(nowTime.AddDate(5, 5, 20)), "Date filter should not pass any value if it is not between the start and end")
}

func TestNewDateFilterEndDateIsEndOfDay(t *testing.T) {
	assertion := require.New(t)
	dFilter := NewDateFilter("", "2024-01-31")
	assertion.True(dFilter.IsEndValid)
	// End should be 2024-01-31T23:59:59.999999999Z, not midnight
	assertion.Equal(2024, dFilter.End.Year())
	assertion.Equal(time.January, dFilter.End.Month())
	assertion.Equal(31, dFilter.End.Day())
	assertion.Equal(23, dFilter.End.Hour())
	assertion.Equal(59, dFilter.End.Minute())
	assertion.Equal(59, dFilter.End.Second())
}

func TestIsPassingDateFilterEndDateIncludesFullDay(t *testing.T) {
	assertion := require.New(t)
	endDate := "2024-01-31"
	dFilter := NewDateFilter("2024-01-01", endDate)

	// A time in the middle of the endDate day should pass
	midDay := time.Date(2024, 1, 31, 15, 30, 0, 0, time.UTC)
	assertion.True(dFilter.IsPassing(midDay), "Date filter should include times during the endDate day")

	// End of the endDate day should pass
	endOfDay := time.Date(2024, 1, 31, 23, 59, 59, 999999999, time.UTC)
	assertion.True(dFilter.IsPassing(endOfDay), "Date filter should include the last nanosecond of endDate")

	// One nanosecond into the next day should fail
	firstNsNextDay := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	assertion.False(dFilter.IsPassing(firstNsNextDay), "Date filter should exclude the start of the day after endDate")
}
