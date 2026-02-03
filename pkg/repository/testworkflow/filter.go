package testworkflow

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type FilterImpl struct {
	FName           string
	FNames          []string
	FLastNDays      int
	FStartDate      *time.Time
	FEndDate        *time.Time
	FStatuses       []testkube.TestWorkflowStatus
	FPage           int
	FPageSize       int
	FSkip           *int
	FTextSearch     string
	FSelector       string
	FTagSelector    string
	FLabelSelector  *LabelSelector
	FActorName      string
	FActorType      testkube.TestWorkflowRunningContextActorType
	FGroupID        string
	FRunnerID       string
	FInitialized    *bool
	FAssigned       *bool
	FHealthRanges   [][2]float64
	FSilentModeFilter *SilentModeFilter
}

type SilentModeFilter string

const (
	SilentModeFilterAll     SilentModeFilter = "all"
	SilentModeFilterOnly    SilentModeFilter = "only"
	SilentModeFilterExclude SilentModeFilter = "exclude"
)

func NewExecutionsFilter() *FilterImpl {
	result := FilterImpl{FPage: 0, FPageSize: PageDefaultLimit}
	return &result
}

func (f *FilterImpl) WithName(name string) *FilterImpl {
	f.FName = name
	return f
}

func (f *FilterImpl) WithNames(names []string) *FilterImpl {
	f.FNames = names
	return f
}

func (f *FilterImpl) WithLastNDays(days int) *FilterImpl {
	f.FLastNDays = days
	return f
}

func (f *FilterImpl) WithStartDate(date time.Time) *FilterImpl {
	f.FStartDate = &date
	return f
}

func (f *FilterImpl) WithEndDate(date time.Time) *FilterImpl {
	f.FEndDate = &date
	return f
}

func (f *FilterImpl) WithStatus(status string) *FilterImpl {
	statuses, err := testkube.ParseTestWorkflowStatusList(status, ",")
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

func (f *FilterImpl) WithSkip(skip int) *FilterImpl {
	f.FSkip = &skip
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

func (f *FilterImpl) WithTagSelector(tagSelector string) *FilterImpl {
	f.FTagSelector = tagSelector
	return f
}

func (f *FilterImpl) WithActorName(actorName string) *FilterImpl {
	f.FActorName = actorName
	return f
}

func (f *FilterImpl) WithActorType(actorType testkube.TestWorkflowRunningContextActorType) *FilterImpl {
	f.FActorType = actorType
	return f
}

func (f *FilterImpl) WithLabelSelector(selector *LabelSelector) *FilterImpl {
	f.FLabelSelector = selector
	return f
}

func (f *FilterImpl) WithGroupID(groupID string) *FilterImpl {
	f.FGroupID = groupID
	return f
}

func (f *FilterImpl) WithRunnerID(runnerID string) *FilterImpl {
	f.FRunnerID = runnerID
	return f
}

func (f *FilterImpl) WithInitialized(initialized bool) *FilterImpl {
	f.FInitialized = &initialized
	return f
}

func (f *FilterImpl) WithAssigned(assigned bool) *FilterImpl {
	f.FAssigned = &assigned
	return f
}

func (f *FilterImpl) WithHealthRanges(ranges [][2]float64) *FilterImpl {
	f.FHealthRanges = ranges
	return f
}

func (f *FilterImpl) WithSilentModeFilter(filter SilentModeFilter) *FilterImpl {
	f.FSilentModeFilter = &filter
	return f
}

func (f FilterImpl) Name() string {
	return f.FName
}

func (f FilterImpl) NameDefined() bool {
	return f.FName != ""
}

func (f FilterImpl) Names() []string {
	return f.FNames
}

func (f FilterImpl) NamesDefined() bool {
	return len(f.FNames) > 0
}

func (f FilterImpl) LastNDaysDefined() bool {
	return f.FLastNDays > 0
}

func (f FilterImpl) LastNDays() int {
	return f.FLastNDays
}

func (f FilterImpl) StartDateDefined() bool {
	return f.FStartDate != nil
}

func (f FilterImpl) StartDate() time.Time {
	return *f.FStartDate
}

func (f FilterImpl) EndDateDefined() bool {
	return f.FEndDate != nil
}

func (f FilterImpl) EndDate() time.Time {
	return *f.FEndDate
}

func (f FilterImpl) StatusesDefined() bool {
	return len(f.FStatuses) != 0
}

func (f FilterImpl) Statuses() []testkube.TestWorkflowStatus {
	return f.FStatuses
}

func (f FilterImpl) Page() int {
	return f.FPage
}

func (f FilterImpl) PageSize() int {
	return f.FPageSize
}

func (f FilterImpl) Skip() int {
	if f.FSkip == nil {
		return 0
	}
	return *f.FSkip
}

func (f FilterImpl) SkipDefined() bool {
	return f.FSkip != nil
}

func (f FilterImpl) TextSearchDefined() bool {
	return f.FTextSearch != ""
}

func (f FilterImpl) TextSearch() string {
	return f.FTextSearch
}

func (f FilterImpl) Selector() string {
	return f.FSelector
}

func (f FilterImpl) TagSelector() string {
	return f.FTagSelector
}

func (f FilterImpl) LabelSelector() *LabelSelector {
	return f.FLabelSelector
}

func (f FilterImpl) ActorName() string {
	return f.FActorName
}

func (f FilterImpl) ActorType() testkube.TestWorkflowRunningContextActorType {
	return f.FActorType
}

func (f FilterImpl) ActorNameDefined() bool {
	return f.FActorName != ""
}

func (f FilterImpl) ActorTypeDefined() bool {
	return f.FActorType != ""
}

func (f FilterImpl) GroupIDDefined() bool {
	return f.FGroupID != ""
}

func (f FilterImpl) GroupID() string {
	return f.FGroupID
}

func (f FilterImpl) RunnerIDDefined() bool {
	return f.FRunnerID != ""
}

func (f FilterImpl) RunnerID() string {
	return f.FRunnerID
}

func (f FilterImpl) InitializedDefined() bool {
	return f.FInitialized != nil
}

func (f FilterImpl) Initialized() bool {
	if f.FInitialized == nil {
		return false
	}
	return *f.FInitialized
}

func (f FilterImpl) AssignedDefined() bool {
	return f.FAssigned != nil
}

func (f FilterImpl) Assigned() bool {
	if f.FAssigned == nil {
		return false
	}
	return *f.FAssigned
}

func (f FilterImpl) HealthRangesDefined() bool {
	return len(f.FHealthRanges) > 0
}

func (f FilterImpl) HealthRanges() [][2]float64 {
	return f.FHealthRanges
}

func (f FilterImpl) SilentModeFilterDefined() bool {
	return f.FSilentModeFilter != nil
}

func (f FilterImpl) SilentModeFilter() SilentModeFilter {
	if f.FSilentModeFilter == nil {
		return SilentModeFilterAll
	}
	return *f.FSilentModeFilter
}
