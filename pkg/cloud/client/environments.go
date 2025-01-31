package client

import (
	"github.com/kubeshop/testkube/pkg/http"
)

func NewEnvironmentsClient(baseUrl, token, orgID string) *EnvironmentsClient {
	return &EnvironmentsClient{
		RESTClient: RESTClient[Environment]{
			BaseUrl: baseUrl,
			Path:    "/organizations/" + orgID + "/environments",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type Environment struct {
	Name              string `json:"name"`
	Id                string `json:"id"`
	Slug              string `json:"slug,omitempty"`
	Connected         bool   `json:"connected"`
	Owner             string `json:"owner"`
	InstallCommand    string `json:"installCommand,omitempty"`
	InstallCommandCli string `json:"installCommandCli,omitempty"`
	OrganizationId    string `json:"organizationId,omitempty"`
	AgentToken        string `json:"agentToken,omitempty"`
	CloudStorage      bool   `json:"cloudStorage,omitempty"`
	NewArchitecture   bool   `json:"newArchitecture,omitempty"`
}

type EnvironmentsClient struct {
	RESTClient[Environment]
}

func (c EnvironmentsClient) Create(env Environment) (Environment, error) {
	return c.RESTClient.Create(env, "/organizations/"+env.Owner+"/environments")
}
