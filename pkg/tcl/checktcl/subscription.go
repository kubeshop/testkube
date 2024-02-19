// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package checktcl

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/pkg/errors"
)

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

type SubscriptionChecker struct {
	proContext config.ProContext
	orgPlan    *OrganizationPlan
}

// NewSubscriptionChecker creates a new subscription checker
func NewSubscriptionChecker(proContext config.ProContext) (*SubscriptionChecker, error) {
	resp, err := http.Get(fmt.Sprintf("%s/organizations/%s/plan", proContext.URL, proContext.OrgID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get organization plan: %d", resp.StatusCode)
	}

	var subscription OrganizationPlan
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		return nil, errors.Wrap(err, "failed to decode organization plan")
	}

	return &SubscriptionChecker{proContext: proContext, orgPlan: &subscription}, nil
}

// GetCurrentOrganizationPlan returns current organization plan
func (c *SubscriptionChecker) GetCurrentOrganizationPlan() (*OrganizationPlan, error) {
	if c.orgPlan == nil {
		return nil, errors.New("organization plan is not set")
	}
	return c.orgPlan, nil
}

// IsOrgPlanEnterprise checks if organization plan is enterprise
func (c *SubscriptionChecker) IsOrgPlanEnterprise() (bool, error) {
	if c.orgPlan == nil {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.TestkubeMode == OrganizationPlanTestkubeModeEnterprise, nil
}

// IsOrgPlanCloud checks if organization plan is cloud
func (c *SubscriptionChecker) IsOrgPlanPro() (bool, error) {
	if c.orgPlan == nil {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.TestkubeMode == OrganizationPlanTestkubeModePro, nil
}

// IsOrgPlanActive checks if organization plan is active
func (c *SubscriptionChecker) IsOrgPlanActive() (bool, error) {
	if c.orgPlan == nil {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.PlanStatus == PlanStatusActive, nil
}
