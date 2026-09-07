package common

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Environment struct {
	Id   string
	Name string
}

func GetEnvironments(url, token, orgID string, skipTLS ...bool) ([]cloudclient.Environment, error) {
	c := cloudclient.NewEnvironmentsClient(url, token, orgID, skipTLS...)
	return c.List()
}

func GetEnvNames(envs []cloudclient.Environment) []string {
	var names []string
	for _, env := range envs {
		names = append(names, env.Name)
	}
	return names
}

func FindEnvID(envs []cloudclient.Environment, name string) string {
	for _, env := range envs {
		if env.Name == name {
			return env.Id
		}
	}
	return ""
}

// ResolveEnvID maps an environment name to its id within the given organization.
//
// Names are matched first, exactly and case-sensitively, mirroring the
// interactive selector. A name that matches no environment is then retried
// against the slug, which is the URL-safe identifier the Control Plane derives
// from the name and which is what users see in dashboard links. Ambiguity is
// an error rather than an arbitrary pick, for the same reason as ResolveOrgID.
func ResolveEnvID(url, token, orgID, name string, skipTLS ...bool) (string, error) {
	envs, err := GetEnvironments(url, token, orgID, skipTLS...)
	if err != nil {
		return "", fmt.Errorf("failed to get environments: %w", err)
	}

	matched := matchEnvs(envs, func(env cloudclient.Environment) bool { return env.Name == name })
	if len(matched) == 0 {
		matched = matchEnvs(envs, func(env cloudclient.Environment) bool {
			return env.Slug != "" && env.Slug == name
		})
	}

	if len(matched) == 0 {
		if len(envs) == 0 {
			return "", fmt.Errorf("no environments available, please create one first")
		}
		return "", fmt.Errorf("no environment named %q, available environments: %s",
			name, strings.Join(GetEnvNames(envs), ", "))
	}

	if len(matched) > 1 {
		ids := make([]string, 0, len(matched))
		for _, env := range matched {
			ids = append(ids, env.Id)
		}
		return "", fmt.Errorf("environment name %q is ambiguous, it matches %d environments (%s), use --env-id to select one",
			name, len(matched), strings.Join(ids, ", "))
	}

	return matched[0].Id, nil
}

func matchEnvs(envs []cloudclient.Environment, match func(cloudclient.Environment) bool) []cloudclient.Environment {
	var matched []cloudclient.Environment
	for _, env := range envs {
		if match(env) {
			matched = append(matched, env)
		}
	}
	return matched
}

func UiGetEnvironmentID(url, token, orgID string, skipTLS ...bool) (string, string, error) {
	// Choose environment from orgs available
	envs, err := GetEnvironments(url, token, orgID, skipTLS...)
	if err != nil {
		return "", "", fmt.Errorf("failed to get environments: %s", err.Error())
	}

	if len(envs) == 0 {
		return "", "", fmt.Errorf("no environments available, please create one first")
	}

	envNames := GetEnvNames(envs)
	envName := ui.Select("Choose environment", envNames)
	envID := FindEnvID(envs, envName)

	return envID, envName, nil
}
