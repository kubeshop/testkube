package client

import (
	"github.com/kubeshop/testkube/pkg/http"
)

func NewEnvironmentsClient(rootDomain, token string) *EnvironmentsClient {
	return &EnvironmentsClient{
		RESTClient: RESTClient[Environment]{
			BaseUrl: "https://api." + rootDomain,
			Path:    "/environments",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type Environment struct {
	Name              string `json:"name"`
	Id                string `json:"id"`
	Connected         bool   `json:"connected"`
	Owner             string `json:"owner"`
	InstallCommand    string `json:"installCommand,omitempty"`
	InstallCommandCli string `json:"installCommandCli,omitempty"`
	OrganizationId    string `json:"organizationId,omitempty"`
	AgentToken        string `json:"agentToken,omitempty"`
}

type EnvironmentsClient struct {
	RESTClient[Environment]
}

func (c EnvironmentsClient) Create(env Environment) (Environment, error) {
	return c.RESTClient.Create(env, "/organizations/"+env.Owner+"/environments")
}
