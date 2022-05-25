package oauth

import "golang.org/x/oauth2"

// ProviderType is provider type
type ProviderType string

const (
	// GithubProviderType is github provider type
	GithubProviderType ProviderType = "github"
)

// Validator describes ouath validation methods
type Validator interface {
	// Validate validates oauth token
	Validate(accessToken string) error
	// GetEndpoint returns oauth endpoint
	GetEndpoint() oauth2.Endpoint
}
