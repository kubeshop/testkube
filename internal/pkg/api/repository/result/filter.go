package result

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type filter struct {
	testName   string
	startDate  *time.Time
	endDate    *time.Time
	lastNDays  int
	statuses   testkube.ExecutionStatuses
	page       int
	pageSize   int
	textSearch string
	selector   string
	objectType string
}

func NewExecutionsFilter() *filter {
	result := filter{page: 0, pageSize: PageDefaultLimit}
	return &result
}

func (f *filter) WithTestName(testName string) *filter {
	f.testName = testName
	return f
}

func (f *filter) WithStartDate(date time.Time) *filter {
	f.startDate = &date
	return f
}

func (f *filter) WithLastNDays(days int) *filter {
	f.lastNDays = days
	return f
}

func (f *filter) WithEndDate(date time.Time) *filter {
	f.endDate = &date
	return f
}

func (f *filter) WithStatus(status string) *filter {
	statuses, err := testkube.ParseExecutionStatusList(status, ",")
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

func (f *filter) WithType(objectType string) *filter {
	f.objectType = objectType
	return f
}
func (f filter) TestName() string {
	return f.testName
}

func (f filter) TestNameDefined() bool {
	return f.testName != ""
}

func (f filter) LastNDaysDefined() bool {
	return f.lastNDays > 0
}

func (f filter) StartDateDefined() bool {
	return f.startDate != nil
}

func (f filter) LastNDays() int {
	return f.lastNDays
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

func (f filter) Statuses() testkube.ExecutionStatuses {
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

func (f filter) TypeDefined() bool {
	return f.objectType != ""
}

func (f filter) Type() string {
	return f.objectType
}

func (f filter) Selector() string {
	return f.selector
}
