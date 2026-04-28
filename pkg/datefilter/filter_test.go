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

// --- RFC 3339 timestamp tests ---

func TestIsPassingDateFilterRFC3339IncludesExactStartTime(t *testing.T) {
	assertion := require.New(t)
	// The exact start timestamp must be included (start bound is inclusive).
	dFilter := NewDateFilter("2024-01-15T13:00:00Z", "2024-01-15T16:00:00Z")
	exactStart := time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC)
	assertion.True(dFilter.IsPassing(exactStart), "exact start time should be included in the range")
}

func TestNewDateFilterRFC3339StartDate(t *testing.T) {
	assertion := require.New(t)
	dFilter := NewDateFilter("2024-01-15T13:00:00Z", "")
	assertion.True(dFilter.IsStartValid, "RFC3339 startDate should be valid")
	assertion.Equal(2024, dFilter.Start.Year())
	assertion.Equal(time.January, dFilter.Start.Month())
	assertion.Equal(15, dFilter.Start.Day())
	assertion.Equal(13, dFilter.Start.Hour())
	assertion.Equal(0, dFilter.Start.Minute())
}

func TestNewDateFilterRFC3339EndDateNoAutoAdvance(t *testing.T) {
	assertion := require.New(t)
	// When a full RFC3339 timestamp is provided for endDate, the exact time
	// must be used – the end-of-day auto-advance must NOT apply.
	dFilter := NewDateFilter("", "2024-01-31T16:00:00Z")
	assertion.True(dFilter.IsEndValid, "RFC3339 endDate should be valid")
	assertion.Equal(16, dFilter.End.Hour(), "hour must be exactly as supplied")
	assertion.Equal(0, dFilter.End.Minute())
	// Must NOT have been advanced to 23:59:59
	assertion.Equal(0, dFilter.End.Second())
}

func TestIsPassingDateFilterRFC3339Range(t *testing.T) {
	assertion := require.New(t)
	// Range: 2024-01-15 13:00 UTC  →  2024-01-15 16:00 UTC
	dFilter := NewDateFilter("2024-01-15T13:00:00Z", "2024-01-15T16:00:00Z")

	before := time.Date(2024, 1, 15, 12, 59, 59, 0, time.UTC)
	inside := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	exactEnd := time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)
	after := time.Date(2024, 1, 15, 16, 0, 1, 0, time.UTC)

	assertion.False(dFilter.IsPassing(before), "time before start should not pass")
	assertion.True(dFilter.IsPassing(inside), "time inside range should pass")
	assertion.True(dFilter.IsPassing(exactEnd), "exact end time should pass")
	assertion.False(dFilter.IsPassing(after), "time after end should not pass")
}

func TestNewDateFilterMixedFormats(t *testing.T) {
	assertion := require.New(t)
	// startDate as date-only, endDate as RFC3339
	dFilter := NewDateFilter("2024-01-15", "2024-01-15T16:00:00Z")
	assertion.True(dFilter.IsStartValid)
	assertion.True(dFilter.IsEndValid)
	// Start should be midnight of 2024-01-15
	assertion.Equal(0, dFilter.Start.Hour())
	// End should be exactly 16:00, not advanced to 23:59
	assertion.Equal(16, dFilter.End.Hour())
	assertion.Equal(0, dFilter.End.Second())
}

func TestNewDateFilterRFC3339WithTimezone(t *testing.T) {
	assertion := require.New(t)
	// RFC3339 with non-UTC timezone offset
	dFilter := NewDateFilter("2024-01-15T15:00:00+02:00", "")
	assertion.True(dFilter.IsStartValid, "RFC3339 with timezone offset should be valid")
	// Parsed time should equal 13:00 UTC
	utc := dFilter.Start.UTC()
	assertion.Equal(13, utc.Hour())
}

func TestNewDateFilterRFC3339FractionalSeconds(t *testing.T) {
	assertion := require.New(t)
	// Fractional seconds are valid RFC3339 and are commonly produced by
	// JavaScript Date.toISOString() and similar tooling.
	dFilter := NewDateFilter("2024-01-15T13:00:00.123Z", "2024-01-15T16:00:00.999Z")
	assertion.True(dFilter.IsStartValid, "RFC3339 startDate with fractional seconds should be valid")
	assertion.True(dFilter.IsEndValid, "RFC3339 endDate with fractional seconds should be valid")
	assertion.Equal(13, dFilter.Start.Hour())
	assertion.Equal(123000000, dFilter.Start.Nanosecond())
	assertion.Equal(16, dFilter.End.Hour())
	// Must NOT have been advanced to 23:59
	assertion.Equal(0, dFilter.End.Second())
}
