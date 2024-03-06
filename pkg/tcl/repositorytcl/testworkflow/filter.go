// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type FilterImpl struct {
	FName       string
	FLastNDays  int
	FStartDate  *time.Time
	FEndDate    *time.Time
	FStatuses   []testkube.TestWorkflowStatus
	FPage       int
	FPageSize   int
	FTextSearch string
	FSelector   string
}

func NewExecutionsFilter() *FilterImpl {
	result := FilterImpl{FPage: 0, FPageSize: PageDefaultLimit}
	return &result
}

func (f *FilterImpl) WithName(name string) *FilterImpl {
	f.FName = name
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

func (f *FilterImpl) WithTextSearch(textSearch string) *FilterImpl {
	f.FTextSearch = textSearch
	return f
}

func (f *FilterImpl) WithSelector(selector string) *FilterImpl {
	f.FSelector = selector
	return f
}

func (f FilterImpl) Name() string {
	return f.FName
}

func (f FilterImpl) NameDefined() bool {
	return f.FName != ""
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

func (f FilterImpl) TextSearchDefined() bool {
	return f.FTextSearch != ""
}

func (f FilterImpl) TextSearch() string {
	return f.FTextSearch
}

func (f FilterImpl) Selector() string {
	return f.FSelector
}
