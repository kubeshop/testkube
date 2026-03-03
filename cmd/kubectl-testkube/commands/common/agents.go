package common

import (
	"fmt"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

func GetAgents(url, token, orgID string, agentType string, includeDeleted bool) ([]cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	query := map[string]string{"includeDeleted": fmt.Sprintf("%v", includeDeleted)}
	if agentType != "" {
		query["type"] = agentType
	}
	return c.ListWithQuery(query)
}

func GetAgent(url, token, orgID string, idOrName string) (cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.Get(idOrName)
}

func GetAgentSecretKey(url, token, orgID string, idOrName string) (string, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.GetSecretKey(idOrName)
}

func DeleteAgent(url, token, orgID string, agentID string) error {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.Delete(agentID)
}

func CreateAgent(url, token, orgID string, agent cloudclient.AgentInput) (cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.Create(agent)
}

func UpdateAgent(url, token, orgID string, idOrName string, agent cloudclient.AgentInput) error {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.Patch(idOrName, agent)
}

func RegenerateAgentSecretKey(url, token, orgID string, idOrName string, gracePeriod string) (cloudclient.RegenerateSecretKeyResponse, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	return c.RegenerateSecretKey(idOrName, gracePeriod)
}
