package cloud

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Environment struct {
	Id   string
	Name string
}

func getEnvironments(rootDomain, token, orgID string) ([]cloudclient.Environment, error) {
	c := cloudclient.NewEnvironmentsClient(rootDomain, token, orgID)
	return c.List()
}

func getEnvNames(envs []cloudclient.Environment) []string {
	var names []string
	for _, env := range envs {
		names = append(names, env.Name)
	}
	return names
}

func findEnvID(envs []cloudclient.Environment, name string) string {
	for _, env := range envs {
		if env.Name == name {
			return env.Id
		}
	}
	return ""
}

func uiGetEnvironmentID(rootDomain, token, orgID string) (string, string, error) {
	// Choose organization from orgs available
	envs, err := getEnvironments(rootDomain, token, orgID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get environments: %s", err.Error())
	}

	if len(envs) == 0 {
		return "", "", fmt.Errorf("no environments available, please create one first")
	}

	envNames := getEnvNames(envs)
	envName := ui.Select("Choose organization", envNames)
	envID := findEnvID(envs, envName)

	return envID, envName, nil
}
