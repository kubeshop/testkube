package testresult

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type filter struct {
	name       string
	lastNDays  int
	startDate  *time.Time
	endDate    *time.Time
	statuses   testkube.TestSuiteExecutionStatuses
	page       int
	pageSize   int
	textSearch string
	selector   string
}

func NewExecutionsFilter() *filter {
	result := filter{page: 0, pageSize: PageDefaultLimit}
	return &result
}

func (f *filter) WithName(name string) *filter {
	f.name = name
	return f
}

func (f *filter) WithLastNDays(days int) *filter {
	f.lastNDays = days
	return f
}

func (f *filter) WithStartDate(date time.Time) *filter {
	f.startDate = &date
	return f
}

func (f *filter) WithEndDate(date time.Time) *filter {
	f.endDate = &date
	return f
}

func (f *filter) WithStatus(status string) *filter {
	statuses, err := testkube.ParseTestSuiteExecutionStatusList(status, ",")
	if err == nil {
		f.statuses = statuses
	}

	return f
}

func (f *filter) WithPage(page int) *filter {
	f.page = page
	return f
}

func (f *filter) WithPageSize(pageSize int) *filter {
	f.pageSize = pageSize
	return f
}

func (f *filter) WithTextSearch(textSearch string) *filter {
	f.textSearch = textSearch
	return f
}

func (f *filter) WithSelector(selector string) *filter {
	f.selector = selector
	return f
}

func (f filter) Name() string {
	return f.name
}

func (f filter) NameDefined() bool {
	return f.name != ""
}

func (f filter) LastNDaysDefined() bool {
	return f.lastNDays > 0
}

func (f filter) LastNDays() int {
	return f.lastNDays
}

func (f filter) StartDateDefined() bool {
	return f.startDate != nil
}

func (f filter) StartDate() time.Time {
	return *f.startDate
}

func (f filter) EndDateDefined() bool {
	return f.endDate != nil
}

func (f filter) EndDate() time.Time {
	return *f.endDate
}

func (f filter) StatusesDefined() bool {
	return len(f.statuses) != 0
}

func (f filter) Statuses() testkube.TestSuiteExecutionStatuses {
	return f.statuses
}

func (f filter) Page() int {
	return f.page
}

func (f filter) PageSize() int {
	return f.pageSize
}

func (f filter) TextSearchDefined() bool {
	return f.textSearch != ""
}

func (f filter) TextSearch() string {
	return f.textSearch
}

func (f filter) Selector() string {
	return f.selector
}
