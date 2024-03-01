// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package checktcl

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionChecker_GetCurrentOrganizationPlan(t *testing.T) {
	tests := []struct {
		name    string
		orgPlan OrganizationPlan
		want    OrganizationPlan
		wantErr bool
	}{
		{
			name:    "Org plan does not exist",
			wantErr: true,
		},
		{
			name: "Org plan exists",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
				IsTrial:      false,
				PlanStatus:   PlanStatusActive,
			},
			want: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
				IsTrial:      false,
				PlanStatus:   PlanStatusActive,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SubscriptionChecker{
				orgPlan: tt.orgPlan,
			}
			got, err := c.GetCurrentOrganizationPlan()
			if (err != nil) != tt.wantErr {
				t.Errorf("SubscriptionChecker.GetCurrentOrganizationPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubscriptionChecker.GetCurrentOrganizationPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionChecker_IsOrgPlanEnterprise(t *testing.T) {
	tests := []struct {
		name    string
		orgPlan OrganizationPlan
		want    bool
		wantErr bool
	}{
		{
			name:    "no org plan",
			wantErr: true,
		},
		{
			name: "enterprise org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pro org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModePro,
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SubscriptionChecker{
				orgPlan: tt.orgPlan,
			}
			got, err := c.IsOrgPlanEnterprise()
			if (err != nil) != tt.wantErr {
				t.Errorf("SubscriptionChecker.IsOrgPlanEnterprise() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SubscriptionChecker.IsOrgPlanEnterprise() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionChecker_IsOrgPlanPro(t *testing.T) {
	tests := []struct {
		name    string
		orgPlan OrganizationPlan
		want    bool
		wantErr bool
	}{
		{
			name:    "no org plan",
			wantErr: true,
		},
		{
			name: "enterprise org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "pro org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModePro,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SubscriptionChecker{
				orgPlan: tt.orgPlan,
			}
			got, err := c.IsOrgPlanPro()
			if (err != nil) != tt.wantErr {
				t.Errorf("SubscriptionChecker.IsOrgPlanPro() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SubscriptionChecker.IsOrgPlanPro() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionChecker_IsOrgPlanActive(t *testing.T) {
	tests := []struct {
		name    string
		orgPlan OrganizationPlan
		want    bool
		wantErr bool
	}{
		{
			name:    "no org plan",
			wantErr: true,
		},
		{
			name: "active org plan",
			orgPlan: OrganizationPlan{
				PlanStatus: PlanStatusActive,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "inactive org plan",
			orgPlan: OrganizationPlan{
				PlanStatus: PlanStatusUnpaid,
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SubscriptionChecker{
				orgPlan: tt.orgPlan,
			}
			got, err := c.IsOrgPlanActive()
			if (err != nil) != tt.wantErr {
				t.Errorf("SubscriptionChecker.IsOrgPlanActive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SubscriptionChecker.IsOrgPlanActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionChecker_IsActiveOrgPlanEnterpriseForFeature(t *testing.T) {
	featureName := "feature"
	tests := []struct {
		name    string
		orgPlan OrganizationPlan
		err     error
	}{
		{
			name: "enterprise active org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
				IsTrial:      false,
				PlanStatus:   PlanStatusActive,
			},
			err: nil,
		},
		{
			name: "no org plan",
			err:  fmt.Errorf("%s is a commercial feature: organization plan is not set", featureName),
		},
		{
			name: "enterprise inactive org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModeEnterprise,
				IsTrial:      false,
				PlanStatus:   PlanStatusUnpaid,
			},
			err: fmt.Errorf("%s is not available: inactive subscription plan", featureName),
		},
		{
			name: "non enterprise actibe org plan",
			orgPlan: OrganizationPlan{
				TestkubeMode: OrganizationPlanTestkubeModePro,
				IsTrial:      false,
				PlanStatus:   PlanStatusActive,
			},
			err: fmt.Errorf("%s is not allowed: wrong subscription plan", featureName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SubscriptionChecker{
				orgPlan: tt.orgPlan,
			}

			err := c.IsActiveOrgPlanEnterpriseForFeature(featureName)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
