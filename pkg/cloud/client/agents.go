package client

import (
	"time"

	"github.com/kubeshop/testkube/pkg/http"
)

const (
	AgentRunnerType = "run"
	AgentGitOpsType = "sync"
)

func NewAgentsClient(baseUrl, token, orgID string) *AgentsClient {
	return &AgentsClient{
		RESTClient: RESTClient[AgentInput, Agent]{
			BaseUrl: baseUrl,
			Path:    "/organizations/" + orgID + "/agents",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type AgentInput struct {
	Name         string            `json:"name"`
	Disabled     bool              `json:"disabled"`
	Type         string            `json:"type"`
	Labels       map[string]string `json:"labels"`
	Environments []string          `json:"environments"`
}

type Agent struct {
	// The unique identifier for the agent.
	ID string `json:"id"`
	// The unique name for the agent.
	Name      string `json:"name"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	// Is the Agent disabled?.
	Disabled     bool               `json:"disabled"`
	Type         string             `json:"type"`
	Labels       map[string]string  `json:"labels"`
	Environments []AgentEnvironment `json:"environments"`
	AccessedAt   *time.Time         `json:"accessedAt,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
	SecretKey    string             `json:"secretKey"`
}

type AgentEnvironment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AgentsClient struct {
	RESTClient[AgentInput, Agent]
}

func (c AgentsClient) CreateRunner(envId string, name string, labels map[string]string) (Agent, error) {
	agent := AgentInput{
		Environments: []string{envId},
		Name:         name,
		Type:         AgentRunnerType,
		Labels:       labels,
	}
	return c.RESTClient.Create(agent)
}

func (c AgentsClient) CreateGitOpsAgent(envId string, name string, labels map[string]string) (Agent, error) {
	agent := AgentInput{
		Environments: []string{envId},
		Name:         name,
		Type:         AgentGitOpsType,
		Labels:       labels,
	}
	return c.RESTClient.Create(agent)
}
