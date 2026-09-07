package common

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Organization struct {
	Id   string
	Name string
}

func GetOrganizations(url, token string, skipTLS ...bool) ([]cloudclient.Organization, error) {
	c := cloudclient.NewOrganizationsClient(url, token, skipTLS...)
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

// ResolveOrgID maps an organization name to its id.
//
// The match is exact and case-sensitive, mirroring what the interactive
// selector does once a name is picked. Unlike FindOrgId it never falls back to
// an empty id: a name that matches nothing, or more than one organization,
// is an error naming what the caller can do about it. Organization names are
// display names and carry no uniqueness guarantee, so guessing between two
// equally valid matches would silently log CI into the wrong organization.
func ResolveOrgID(url, token, name string, skipTLS ...bool) (string, error) {
	orgs, err := GetOrganizations(url, token, skipTLS...)
	if err != nil {
		return "", fmt.Errorf("failed to get organizations: %w", err)
	}

	var matched []cloudclient.Organization
	for _, org := range orgs {
		if org.Name == name {
			matched = append(matched, org)
		}
	}

	if len(matched) == 0 {
		if len(orgs) == 0 {
			return "", fmt.Errorf("no organizations available, please create one first")
		}
		return "", fmt.Errorf("no organization named %q, available organizations: %s",
			name, strings.Join(GetOrgNames(orgs), ", "))
	}

	if len(matched) > 1 {
		ids := make([]string, 0, len(matched))
		for _, org := range matched {
			ids = append(ids, org.Id)
		}
		return "", fmt.Errorf("organization name %q is ambiguous, it matches %d organizations (%s), use --org-id to select one",
			name, len(matched), strings.Join(ids, ", "))
	}

	return matched[0].Id, nil
}

func UiGetOrganizationId(url, token string, skipTLS ...bool) (string, string, error) {
	// Choose organization from orgs available
	orgs, err := GetOrganizations(url, token, skipTLS...)
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
