package common

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

type Environment struct {
	Id   string
	Name string
}

func GetEnvironments(url, token, orgID string) ([]cloudclient.Environment, error) {
	c := cloudclient.NewEnvironmentsClient(url, token, orgID)
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

func UiGetEnvironmentID(url, token, orgID string) (string, string, error) {
	// Choose organization from orgs available
	envs, err := GetEnvironments(url, token, orgID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get environments: %s", err.Error())
	}

	if len(envs) == 0 {
		return "", "", fmt.Errorf("no environments available, please create one first")
	}

	envNames := GetEnvNames(envs)
	envName := ui.Select("Choose organization", envNames)
	envID := FindEnvID(envs, envName)

	return envID, envName, nil
}
