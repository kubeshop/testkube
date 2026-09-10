package common

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
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

// ResolveNamedOrg replaces the organization name on master with the id it
// refers to, and does nothing when no name was given. It never prompts, so it
// suits commands that treat a missing organization as valid input rather than
// something to ask about.
//
// Call this before ResolveNamedEnv: the environment lookup is scoped to an
// organization and reads the id this sets. The id and name flags are mutually
// exclusive at the cobra level, so a caller cannot reach this with both set.
func ResolveNamedOrg(apiURL, token string, master *config.Master, skipTLS bool) error {
	if master.OrgName == "" {
		return nil
	}

	orgID, err := ResolveOrgID(apiURL, token, master.OrgName, skipTLS)
	if err != nil {
		return err
	}
	master.OrgId = orgID

	return nil
}

// ResolveOrgOrPrompt determines which organization the command should act on:
// an explicit id wins, then a name resolved against the Control Plane, and only
// when neither is given does the user get an interactive selector.
//
// Unlike ResolveNamedOrg this may prompt, so it suits commands that can ask.
// Resolve the organization before the environment: ResolveEnvOrPrompt needs the
// id this returns to scope its lookup.
func ResolveOrgOrPrompt(apiURL, token string, master config.Master, skipTLS bool) (string, error) {
	if master.OrgId != "" {
		return master.OrgId, nil
	}

	if master.OrgName != "" {
		return ResolveOrgID(apiURL, token, master.OrgName, skipTLS)
	}

	if !selectorInteractive() {
		return "", fmt.Errorf("no organization selected and the terminal is not interactive, pass --org-id or --org-name")
	}

	orgID, _, err := UiGetOrganizationId(apiURL, token, skipTLS)
	return orgID, err
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
