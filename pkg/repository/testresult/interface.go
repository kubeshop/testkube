package testresult

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const PageDefaultLimit int = 100

type Filter interface {
	Name() string
	NameDefined() bool
	LastNDays() int
	LastNDaysDefined() bool
	StartDate() time.Time
	StartDateDefined() bool
	EndDate() time.Time
	EndDateDefined() bool
	Statuses() testkube.TestSuiteExecutionStatuses
	StatusesDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Selector() string
}

type Sequences interface {
	// GetNextExecutionNumber gets next execution number by name
	GetNextExecutionNumber(ctx context.Context, name string) (number int32, err error)
}
