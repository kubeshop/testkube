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

const environmentRegistrationTokenEndpoint = "registration-token/rotate"

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

func (c EnvironmentsClient) rotateRegistrationTokenURL(envID, gracePeriod string) (string, error) {
	path, err := url.JoinPath(c.BaseUrl, c.Path, envID, environmentRegistrationTokenEndpoint)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	if gracePeriod != "" {
		q := u.Query()
		q.Set("gracePeriod", gracePeriod)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func (c EnvironmentsClient) RotateRegistrationToken(envID, gracePeriod string) (RotateRegistrationTokenResponse, error) {
	path, err := c.rotateRegistrationTokenURL(envID, gracePeriod)
	if err != nil {
		return RotateRegistrationTokenResponse{}, err
	}
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, path, nil)
	if err != nil {
		return RotateRegistrationTokenResponse{}, err
	}
	setBearerAuth(req, c.Token)

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
