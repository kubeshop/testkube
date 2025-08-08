package common

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
)

// GetOAuthAccessToken strictly checks for OAuth authentication and returns the access token
// Returns token, error. Error is non-nil if not OAuth authenticated.
func GetOAuthAccessToken() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Strict OAuth checks
	if cfg.ContextType != config.ContextTypeCloud {
		return "", fmt.Errorf("not in cloud context (current: %s)", cfg.ContextType)
	}

	if cfg.CloudContext.TokenType != config.TokenTypeOIDC {
		return "", fmt.Errorf("not OAuth authenticated (current: %s)", cfg.CloudContext.TokenType)
	}

	if cfg.CloudContext.ApiKey == "" {
		return "", fmt.Errorf("no access token available")
	}

	if cfg.CloudContext.RefreshToken == "" {
		return "", fmt.Errorf("no refresh token available - not a valid OAuth session")
	}

	return cfg.CloudContext.ApiKey, nil
}

// RefreshOAuthToken refreshes the OAuth access token if needed and returns the current valid token
// This function handles the entire refresh flow and updates the config
func RefreshOAuthToken() (accessToken string, err error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Strict OAuth validation first
	if cfg.ContextType != config.ContextTypeCloud {
		return "", fmt.Errorf("not in cloud context")
	}

	if cfg.CloudContext.TokenType != config.TokenTypeOIDC {
		return "", fmt.Errorf("not OAuth authenticated")
	}

	if cfg.CloudContext.ApiKey == "" || cfg.CloudContext.RefreshToken == "" {
		return "", fmt.Errorf("missing OAuth tokens")
	}

	// Determine auth URI
	authURI := cfg.CloudContext.AuthUri
	if authURI == "" {
		authURI = fmt.Sprintf("%s/idp", cfg.CloudContext.ApiUri)
	}

	// Try to refresh the token
	newAccessToken, newRefreshToken, err := cloudlogin.CheckAndRefreshToken(
		context.Background(),
		authURI,
		cfg.CloudContext.ApiKey,
		cfg.CloudContext.RefreshToken,
	)
	if err != nil {
		return "", fmt.Errorf("failed to refresh OAuth token: %w", err)
	}

	// Update tokens in config if they changed
	if newAccessToken != cfg.CloudContext.ApiKey || newRefreshToken != cfg.CloudContext.RefreshToken {
		cfg.CloudContext.ApiKey = newAccessToken
		cfg.CloudContext.RefreshToken = newRefreshToken

		if err := config.Save(cfg); err != nil {
			return newAccessToken, fmt.Errorf("token refreshed but failed to save config: %w", err)
		}
	}

	return newAccessToken, nil
}

// IsOAuthAuthenticated returns true only if strictly OAuth authenticated
func IsOAuthAuthenticated() bool {
	_, err := GetOAuthAccessToken()
	return err == nil
}
