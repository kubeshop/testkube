// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package checktcl

// Enterprise / Pro mode.
type OrganizationPlanTestkubeMode string

const (
	OrganizationPlanTestkubeModeEnterprise OrganizationPlanTestkubeMode = "enterprise"
	// TODO: Use "pro" in the future when refactoring TK Pro API server to use "pro" instead of "cloud"
	OrganizationPlanTestkubeModePro OrganizationPlanTestkubeMode = "cloud"
)

// Ref: #/components/schemas/PlanStatus
type PlanStatus string

const (
	PlanStatusActive            PlanStatus = "Active"
	PlanStatusCanceled          PlanStatus = "Canceled"
	PlanStatusIncomplete        PlanStatus = "Incomplete"
	PlanStatusIncompleteExpired PlanStatus = "IncompleteExpired"
	PlanStatusPastDue           PlanStatus = "PastDue"
	PlanStatusTrailing          PlanStatus = "Trailing"
	PlanStatusUnpaid            PlanStatus = "Unpaid"
	PlanStatusDeleted           PlanStatus = "Deleted"
	PlanStatusLocked            PlanStatus = "Locked"
	PlanStatusBlocked           PlanStatus = "Blocked"
)

// Ref: #/components/schemas/OrganizationPlan
type OrganizationPlan struct {
	// Enterprise / Pro mode.
	TestkubeMode OrganizationPlanTestkubeMode `json:"testkubeMode"`
	// Is current plan trial.
	IsTrial    bool       `json:"isTrial"`
	PlanStatus PlanStatus `json:"planStatus"`
}

func (p OrganizationPlan) IsEnterprise() bool {
	return p.TestkubeMode == OrganizationPlanTestkubeModeEnterprise
}

func (p OrganizationPlan) IsPro() bool {
	return p.TestkubeMode == OrganizationPlanTestkubeModePro
}

func (p OrganizationPlan) IsActive() bool {
	return p.PlanStatus == PlanStatusActive
}

func (p OrganizationPlan) IsEmpty() bool {
	return p.PlanStatus == "" && p.TestkubeMode == "" && !p.IsTrial
}

type GetOrganizationPlanRequest struct{}
type GetOrganizationPlanResponse struct {
	TestkubeMode string
	IsTrial      bool
	PlanStatus   string
}
