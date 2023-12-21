package common

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Organization struct {
	Id   string
	Name string
}

func GetOrganizations(url, token string) ([]cloudclient.Organization, error) {
	c := cloudclient.NewOrganizationsClient(url, token)
	return c.List()
}

func GetOrgNames(orgs []cloudclient.Organization) []string {
	var names []string
	for _, org := range orgs {
		names = append(names, org.Name)
	}
	return names
}

func FindOrgId(orgs []cloudclient.Organization, name string) string {
	for _, org := range orgs {
		if org.Name == name {
			return org.Id
		}
	}
	return ""
}

func UiGetOrganizationId(url, token string) (string, string, error) {
	// Choose organization from orgs available
	orgs, err := GetOrganizations(url, token)
	if err != nil {
		return "", "", fmt.Errorf("failed to get organizations: %s", err.Error())
	}

	if len(orgs) == 0 {
		return "", "", fmt.Errorf("no organizations available, please create one first")
	}

	orgNames := GetOrgNames(orgs)
	orgName := ui.Select("Choose organization", orgNames)
	orgId := FindOrgId(orgs, orgName)

	return orgId, orgName, nil
}
