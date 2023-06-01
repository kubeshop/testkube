package cloud

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Organization struct {
	Id   string
	Name string
}

func getOrganizations(rootDomain, token string) ([]cloudclient.Organization, error) {
	c := cloudclient.NewOrganizationsClient(rootDomain, token)
	return c.List()
}

func getNames(orgs []cloudclient.Organization) []string {
	var names []string
	for _, org := range orgs {
		names = append(names, org.Name)
	}
	return names
}

func findId(orgs []cloudclient.Organization, name string) string {
	for _, org := range orgs {
		if org.Name == name {
			return org.Id
		}
	}
	return ""
}

func uiGetOrganizationId(rootDomain, token string) (string, string, error) {
	// Choose organization from orgs available
	orgs, err := getOrganizations(rootDomain, token)
	if err != nil {
		return "", "", fmt.Errorf("failed to get organizations: %s", err.Error())
	}

	if len(orgs) == 0 {
		return "", "", fmt.Errorf("no organizations available, please create one first")
	}

	orgNames := getNames(orgs)
	orgName := ui.Select("Choose organization", orgNames)
	orgId := findId(orgs, orgName)

	return orgId, orgName, nil
}
