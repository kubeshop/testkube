package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"

	"github.com/kubeshop/testkube/pkg/http"
)

func NewEnvironmentsClient(baseUrl, token, orgID string) *EnvironmentsClient {
	return &EnvironmentsClient{
		RESTClient: RESTClient[Environment, Environment]{
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
	RESTClient[Environment, Environment]
}

func (c EnvironmentsClient) Create(env Environment) (Environment, error) {
	return c.RESTClient.Create(env, "/organizations/"+env.Owner+"/environments")
}

func (c EnvironmentsClient) EnableNewArchitecture(env Environment) error {
	path := c.BaseUrl + c.Path + "/" + env.Id
	body := map[string]interface{}{
		"id":              env.Id,
		"name":            env.Name,
		"connected":       env.Connected,
		"cloudStorage":    env.CloudStorage,
		"newArchitecture": true,
	}
	if !env.Connected {
		body["cloudStorage"] = true
	}
	d, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := nethttp.NewRequest("PATCH", path, bytes.NewBuffer(d))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error updating %s: can't read response: %s", c.Path, err)
		}
		return fmt.Errorf("error updating %s: %s", path, d)
	}
	return nil
}
