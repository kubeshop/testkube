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
	"net/http"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

type SubscriptionChecker struct {
	proContext config.ProContext
	orgPlan    *OrganizationPlan
}

// NewCLISubscriptionChecker creates a new subscription checker using a user token instead of the agent token
func NewCLISubscriptionChecker(proContext config.ProContext, token string) (*SubscriptionChecker, error) {
	var bearer = fmt.Sprintf("Bearer %s", token)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/organizations/%s/plan", proContext.URL, proContext.OrgID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
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

// NewAgentSubscriptionChecker creates a new subscription checker using the agent token
func NewAgentSubscriptionChecker(ctx context.Context, proContext config.ProContext, cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn) (*SubscriptionChecker, error) {
	executor := executor.NewCloudGRPCExecutor(cloudClient, grpcConn, proContext.APIKey)

	req := GetOrganizationPlanRequest{}
	response, err := executor.Execute(ctx, cloudconfig.CmdConfigGetOrganizationPlan, req)
	if err != nil {
		return nil, err
	}

	var commandResponse GetOrganizationPlanResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	subscription := OrganizationPlan{
		TestkubeMode: OrganizationPlanTestkubeMode(commandResponse.TestkubeMode),
		IsTrial:      commandResponse.IsTrial,
		PlanStatus:   PlanStatus(commandResponse.PlanStatus),
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
