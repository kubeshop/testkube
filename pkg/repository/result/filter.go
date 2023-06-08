package result

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type FilterImpl struct {
	FTestName   string                     `json:"testName"`
	FStartDate  *time.Time                 `json:"startDate"`
	FEndDate    *time.Time                 `json:"endDate"`
	FLastNDays  int                        `json:"lastNDays"`
	FStatuses   testkube.ExecutionStatuses `json:"statuses"`
	FPage       int                        `json:"page"`
	FPageSize   int                        `json:"pageSize"`
	FTextSearch string                     `json:"textSearch"`
	FSelector   string                     `json:"selector"`
	FObjectType string                     `json:"objectType"`
}

func NewExecutionsFilter() *FilterImpl {
	result := FilterImpl{FPage: 0, FPageSize: PageDefaultLimit}
	return &result
}

func (f *FilterImpl) WithTestName(testName string) *FilterImpl {
	f.FTestName = testName
	return f
}

func (f *FilterImpl) WithStartDate(date time.Time) *FilterImpl {
	f.FStartDate = &date
	return f
}

func (f *FilterImpl) WithLastNDays(days int) *FilterImpl {
	f.FLastNDays = days
	return f
}

func (f *FilterImpl) WithEndDate(date time.Time) *FilterImpl {
	f.FEndDate = &date
	return f
}

func (f *FilterImpl) WithStatus(status string) *FilterImpl {
	statuses, err := testkube.ParseExecutionStatusList(status, ",")
	if err == nil {
		f.FStatuses = statuses
	}

	return f
}

func (f *FilterImpl) WithPage(page int) *FilterImpl {
	f.FPage = page
	return f
}

func (f *FilterImpl) WithPageSize(pageSize int) *FilterImpl {
	f.FPageSize = pageSize
	return f
}

func (f *FilterImpl) WithTextSearch(textSearch string) *FilterImpl {
	f.FTextSearch = textSearch
	return f
}

func (f *FilterImpl) WithSelector(selector string) *FilterImpl {
	f.FSelector = selector
	return f
}

func (f *FilterImpl) WithType(objectType string) *FilterImpl {
	f.FObjectType = objectType
	return f
}

func (f *FilterImpl) TestName() string {
	return f.FTestName
}

func (f *FilterImpl) TestNameDefined() bool {
	return f.FTestName != ""
}

func (f *FilterImpl) LastNDaysDefined() bool {
	return f.FLastNDays > 0
}

func (f *FilterImpl) StartDateDefined() bool {
	return f.FStartDate != nil
}

func (f *FilterImpl) LastNDays() int {
	return f.FLastNDays
}

func (f *FilterImpl) StartDate() time.Time {
	return *f.FStartDate
}

func (f *FilterImpl) EndDateDefined() bool {
	return f.FEndDate != nil
}

func (f *FilterImpl) EndDate() time.Time {
	return *f.FEndDate
}

func (f *FilterImpl) StatusesDefined() bool {
	return len(f.FStatuses) != 0
}

func (f *FilterImpl) Statuses() testkube.ExecutionStatuses {
	return f.FStatuses
}

func (f *FilterImpl) Page() int {
	return f.FPage
}

func (f *FilterImpl) PageSize() int {
	return f.FPageSize
}

func (f *FilterImpl) TextSearchDefined() bool {
	return f.FTextSearch != ""
}

func (f *FilterImpl) TextSearch() string {
	return f.FTextSearch
}

func (f *FilterImpl) TypeDefined() bool {
	return f.FObjectType != ""
}

func (f *FilterImpl) Type() string {
	return f.FObjectType
}

func (f *FilterImpl) Selector() string {
	return f.FSelector
}
