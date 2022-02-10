package result

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type filter struct {
	testName   string
	startDate  *time.Time
	endDate    *time.Time
	status     *testkube.ExecutionStatus
	page       int
	pageSize   int
	textSearch string
	tags       []string
	objectType string
}

func NewExecutionsFilter() *filter {
	result := filter{page: 0, pageSize: PageDefaultLimit}
	return &result
}

func (f *filter) WithScriptName(testName string) *filter {
	f.testName = testName
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

func (f *filter) WithStatus(status testkube.ExecutionStatus) *filter {
	f.status = &status
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

func (f *filter) WithTags(tags []string) *filter {
	f.tags = tags
	return f
}

func (f *filter) WithType(objectType string) *filter {
	f.objectType = objectType
	return f
}
func (f filter) ScriptName() string {
	return f.testName
}

func (f filter) ScriptNameDefined() bool {
	return f.testName != ""
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

func (f filter) StatusDefined() bool {
	return f.status != nil
}

func (f filter) Status() testkube.ExecutionStatus {
	return *f.status
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

func (f filter) Tags() []string {
	return f.tags
}
