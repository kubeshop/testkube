package common

import cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"

func GetAgents(url, token, orgID string, agentType string) ([]cloudclient.Agent, error) {
	c := cloudclient.NewAgentsClient(url, token, orgID)
	if agentType == "" {
		return c.List()
	}
	return c.ListWithQuery(map[string]string{"type": agentType})
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
