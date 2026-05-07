package datefilter

import "time"

const DateFormatISO8601 string = "2006-01-02"

// DateFilter is used to filter dates
type DateFilter struct {
	Start        time.Time
	End          time.Time
	IsStartValid bool
	IsEndValid   bool
}

// NewDateFilter creates new DateFilter with the start date and end date.
// Both parameters accept either a date-only string (YYYY-MM-DD) or a full
// RFC 3339 timestamp (e.g. 2024-01-15T13:00:00Z).
//
// When endDate is provided as a date-only value the filter is automatically
// advanced to the last nanosecond of that day so the entire day is included.
// When endDate is provided as an RFC 3339 timestamp the exact time is used
// without any adjustment.
func NewDateFilter(startDate string, endDate string) DateFilter {
	dFilter := DateFilter{}

	// Parse startDate – try RFC3339Nano (superset of RFC3339, accepts fractional
	// seconds), then RFC3339, then fall back to date-only.
	if t, err := time.Parse(time.RFC3339Nano, startDate); err == nil {
		dFilter.Start = t
		dFilter.IsStartValid = true
	} else if t, err := time.Parse(time.RFC3339, startDate); err == nil {
		dFilter.Start = t
		dFilter.IsStartValid = true
	} else if t, err := time.Parse(DateFormatISO8601, startDate); err == nil {
		dFilter.Start = t
		dFilter.IsStartValid = true
	}

	// Parse endDate – same priority order.
	if t, err := time.Parse(time.RFC3339Nano, endDate); err == nil {
		dFilter.End = t
		dFilter.IsEndValid = true
	} else if t, err := time.Parse(time.RFC3339, endDate); err == nil {
		dFilter.End = t
		dFilter.IsEndValid = true
	} else if t, err := time.Parse(DateFormatISO8601, endDate); err == nil {
		// Date-only: advance to end-of-day so the entire day is included.
		dFilter.End = t.Add(24*time.Hour - time.Nanosecond)
		dFilter.IsEndValid = true
	}

	return dFilter
}

// IsPassing reports whether date falls within the filter's configured range.
// Both the start and end bounds are inclusive.
func (dFilter DateFilter) IsPassing(date time.Time) bool {
	if !dFilter.IsStartValid {
		return true
	}
	if dFilter.Start.After(date) {
		return false
	}
	if !dFilter.IsEndValid {
		return true
	}
	return !dFilter.End.Before(date)
}
