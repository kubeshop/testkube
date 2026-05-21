package common

import (
	"fmt"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

func GetAgents(url, token, orgID string, agentType string, includeDeleted bool, skipTLS ...bool) ([]cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	query := map[string]string{"includeDeleted": fmt.Sprintf("%v", includeDeleted)}
	if agentType != "" {
		query["type"] = agentType
	}
	return c.ListWithQuery(query)
}

func GetAgent(url, token, orgID string, idOrName string, skipTLS ...bool) (cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.Get(idOrName)
}

func GetAgentSecretKey(url, token, orgID string, idOrName string, skipTLS ...bool) (string, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.GetSecretKey(idOrName)
}

func DeleteAgent(url, token, orgID string, agentID string, skipTLS ...bool) error {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.Delete(agentID)
}

func CreateAgent(url, token, orgID string, agent cloudclient.AgentInput, skipTLS ...bool) (cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.Create(agent)
}

func UpdateAgent(url, token, orgID string, idOrName string, agent cloudclient.AgentInput, skipTLS ...bool) error {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.Patch(idOrName, agent)
}

func RegenerateAgentSecretKey(url, token, orgID string, idOrName string, gracePeriod string, skipTLS ...bool) (cloudclient.RegenerateSecretKeyResponse, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID, skipTLS...)
	return c.RegenerateSecretKey(idOrName, gracePeriod)
}
