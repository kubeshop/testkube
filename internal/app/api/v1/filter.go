package v1

import "time"

const DateFormatISO8601 string = "2006-01-02"

// DateFilter is used to filter dates
type DateFilter struct {
	Start        time.Time
	End          time.Time
	IsStartValid bool
	IsEndValid   bool
}

// NewDateFilter creates new DateFilter with the start date and end date
func NewDateFilter(startDate string, endDate string) DateFilter {
	var err error

	dFilter := DateFilter{}
	dFilter.Start, err = time.Parse(DateFormatISO8601, startDate)
	dFilter.IsStartValid = (err == nil)
	dFilter.End, err = time.Parse(DateFormatISO8601, endDate)
	dFilter.IsEndValid = err == nil

	return dFilter
}

// IsPassing tests if a specific date is passing the filter
func (dFilter DateFilter) IsPassing(date time.Time) bool {
	if !dFilter.IsStartValid {
		return true
	}
	if dFilter.Start.Before(date) {
		if !dFilter.IsEndValid {
			return true
		}
		if dFilter.End.After(date) {
			return true
		}
	}
	return false
}
