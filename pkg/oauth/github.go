package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// NewGithubValidator creates new github validator
func NewGithubValidator(client *http.Client, clientID, clientSecret string, scopes []string) *GithubValidator {
	return &GithubValidator{
		client:       client,
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       scopes,
	}
}

// GithubValidator is github oauth validator
type GithubValidator struct {
	client       *http.Client
	clientID     string
	clientSecret string
	scopes       []string
}

// Validate validates oauth token
func (v GithubValidator) Validate(accessToken string) error {
	uri := fmt.Sprintf("https://api.github.com/applications/%s/token", v.clientID)
	data := GithubValidatorRequest{
		AccessToken: accessToken,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return err
	}

	parsedURI.User = url.UserPassword(v.clientID, v.clientSecret)
	req, err := http.NewRequest(http.MethodPost, parsedURI.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status: %s", resp.Status)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var token GithubValidatorResponse
	if err = json.Unmarshal(result, &token); err != nil {
		return err
	}

	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("token expired at %v", token.ExpiresAt)
	}

	scopeMap := make(map[string]struct{}, len(token.Scopes))
	for _, scope := range token.Scopes {
		scopeMap[scope] = struct{}{}
	}

	for _, scope := range v.scopes {
		if _, ok := scopeMap[scope]; !ok {
			return fmt.Errorf("token doesn't contain scope %s", scope)
		}
	}

	return nil
}

// GetEndpoint returns endpoint
func (v GithubValidator) GetEndpoint() oauth2.Endpoint {
	return github.Endpoint
}

// GithubValidatorRequest contains github validation request
type GithubValidatorRequest struct {
	AccessToken string `json:"access_token"`
}

// GithubValidatorResponse contains github validation response
type GithubValidatorResponse struct {
	ExpiresAt *time.Time `json:"expires_at"`
	Scopes    []string   `json:"scopes"`
}
