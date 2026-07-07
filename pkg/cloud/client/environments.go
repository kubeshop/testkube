package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"time"

	"github.com/kubeshop/testkube/pkg/http"
)

func NewEnvironmentsClient(baseUrl, token, orgID string, insecure ...bool) *EnvironmentsClient {
	return &EnvironmentsClient{
		RESTClient: RESTClient[Environment, Environment]{
			BaseUrl: baseUrl,
			Path:    "/organizations/" + orgID + "/environments",
			Client:  http.NewClient(insecure...),
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

type RotateRegistrationTokenResponse struct {
	RegistrationToken string    `json:"registrationToken"`
	GracePeriod       string    `json:"gracePeriod"`
	OldTokenExpiresAt time.Time `json:"oldTokenExpiresAt"`
}

func (c EnvironmentsClient) Create(env Environment) (Environment, error) {
	return c.RESTClient.Create(env, "/organizations/"+env.Owner+"/environments")
}

func (c EnvironmentsClient) RotateRegistrationToken(envID, gracePeriod string) (RotateRegistrationTokenResponse, error) {
	path := c.BaseUrl + c.Path + "/" + envID + "/registration-token"
	if gracePeriod != "" {
		path += "?gracePeriod=" + url.QueryEscape(gracePeriod)
	}
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodDelete, path, nil)
	if err != nil {
		return RotateRegistrationTokenResponse{}, err
	}
	req.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return RotateRegistrationTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return RotateRegistrationTokenResponse{}, fmt.Errorf("rotate registration token: can't read response: %w", readErr)
		}
		return RotateRegistrationTokenResponse{}, fmt.Errorf("rotate registration token: %s", body)
	}

	var result RotateRegistrationTokenResponse
	return result, json.NewDecoder(resp.Body).Decode(&result)
}
