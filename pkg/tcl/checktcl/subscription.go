// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package checktcl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

type SubscriptionChecker struct {
	proContext config.ProContext
	orgPlan    OrganizationPlan
}

// NewSubscriptionChecker creates a new subscription checker using the agent token
func NewSubscriptionChecker(ctx context.Context, proContext config.ProContext, cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn) (SubscriptionChecker, error) {
	executor := executor.NewCloudGRPCExecutor(cloudClient, grpcConn, proContext.APIKey)

	req := GetOrganizationPlanRequest{}
	response, err := executor.Execute(ctx, cloudconfig.CmdConfigGetOrganizationPlan, req)
	if err != nil {
		return SubscriptionChecker{}, err
	}

	var commandResponse GetOrganizationPlanResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return SubscriptionChecker{}, err
	}

	subscription := OrganizationPlan{
		TestkubeMode: OrganizationPlanTestkubeMode(commandResponse.TestkubeMode),
		IsTrial:      commandResponse.IsTrial,
		PlanStatus:   PlanStatus(commandResponse.PlanStatus),
	}

	return SubscriptionChecker{proContext: proContext, orgPlan: subscription}, nil
}

// GetCurrentOrganizationPlan returns current organization plan
func (c *SubscriptionChecker) GetCurrentOrganizationPlan() (OrganizationPlan, error) {
	if c.orgPlan.IsEmpty() {
		return OrganizationPlan{}, errors.New("organization plan is not set")
	}
	return c.orgPlan, nil
}

// IsOrgPlanEnterprise checks if organization plan is enterprise
func (c *SubscriptionChecker) IsOrgPlanEnterprise() (bool, error) {
	if c.orgPlan.IsEmpty() {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.IsEnterprise(), nil
}

// IsOrgPlanCloud checks if organization plan is cloud
func (c *SubscriptionChecker) IsOrgPlanPro() (bool, error) {
	if c.orgPlan.IsEmpty() {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.IsPro(), nil
}

// IsOrgPlanActive checks if organization plan is active
func (c *SubscriptionChecker) IsOrgPlanActive() (bool, error) {
	if c.orgPlan.IsEmpty() {
		return false, errors.New("organization plan is not set")
	}
	return c.orgPlan.IsActive(), nil
}

// IsActiveOrgPlanEnterpriseForFeature checks if organization plan is active and enterprise for feature
func (c *SubscriptionChecker) IsActiveOrgPlanEnterpriseForFeature(featureName string) error {
	plan, err := c.GetCurrentOrganizationPlan()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s is a commercial feature", featureName))
	}

	if !plan.IsActive() {
		return errors.New(fmt.Sprintf("%s is not available: inactive subscription plan", featureName))
	}

	if !plan.IsEnterprise() {
		return errors.New(fmt.Sprintf("%s is not allowed: wrong subscription plan", featureName))
	}

	return nil
}
