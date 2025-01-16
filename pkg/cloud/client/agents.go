package client

import (
	"time"

	"github.com/kubeshop/testkube/pkg/http"
)

func NewAgentsClient(baseUrl, token, orgID string) *AgentsClient {
	return &AgentsClient{
		RESTClient: RESTClient[Agent]{
			BaseUrl: baseUrl,
			Path:    "/organizations/" + orgID + "/agents",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type Agent struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Disabled       bool              `json:"disabled,omitempty"`
	Labels         map[string]string `json:"labels"`
	EnvironmentIDs []string          `json:"environments"`
	SecretKey      string            `json:"secretkey,omitempty"`
	AccessedAt     *time.Time        `json:"accessedat,omitempty"`
	CreatedAt      *time.Time        `json:"createdat,omitempty"`

	Type string `json:"type"`
}

type AgentsClient struct {
	RESTClient[Agent]
}

func (c AgentsClient) CreateRunner(envId string, name string, labels map[string]string) (Agent, error) {
	agent := Agent{
		EnvironmentIDs: []string{envId},
		Name:           name,
		Type:           "run",
		Labels:         labels,
	}
	return c.RESTClient.Create(agent)
}
