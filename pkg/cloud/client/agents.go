package client

import (
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"time"

	"github.com/kubeshop/testkube/internal/common"
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
	Name         string             `json:"name,omitempty"`
	Disabled     *bool              `json:"disabled,omitempty"`
	Type         string             `json:"type,omitempty"`
	Labels       *map[string]string `json:"labels,omitempty"`
	Environments []string           `json:"environments,omitempty"`
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

func (c AgentsClient) GetSecretKey(idOrName string) (string, error) {
	path := c.BaseUrl + c.Path + "/" + idOrName + "/secret-key"
	req, err := nethttp.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return "", err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error getting %s: can't read response: %s", c.Path, err)
		}
		return "", fmt.Errorf("error getting %s: %s", path, d)
	}

	var e struct {
		SecretKey string `json:"secretKey"`
	}
	err = json.NewDecoder(resp.Body).Decode(&e)
	return e.SecretKey, err
}

func (c AgentsClient) CreateRunner(envId string, name string, labels map[string]string) (Agent, error) {
	agent := AgentInput{
		Environments: []string{envId},
		Name:         name,
		Type:         AgentRunnerType,
		Labels:       common.Ptr(labels),
	}
	return c.RESTClient.Create(agent)
}

func (c AgentsClient) CreateGitOpsAgent(envId string, name string, labels map[string]string) (Agent, error) {
	agent := AgentInput{
		Environments: []string{envId},
		Name:         name,
		Type:         AgentGitOpsType,
		Labels:       common.Ptr(labels),
	}
	return c.RESTClient.Create(agent)
}
