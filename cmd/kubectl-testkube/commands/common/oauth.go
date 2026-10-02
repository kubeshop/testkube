package common

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/cloudlogin"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

// isUserTokenType reports whether the stored token is a user-login token
// (OIDC or email magic-link).
func isUserTokenType(t string) bool {
	return t == config.TokenTypeOIDC || t == config.TokenTypeEmailLink
}

// GetOAuthAccessToken checks for user-login authentication (OIDC or email-link) and returns the access token.
// Returns token, error. Error is non-nil if not authenticated via either user flow.
func GetOAuthAccessToken() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.ContextType != config.ContextTypeCloud {
		return "", fmt.Errorf("not in cloud context (current: %s)", cfg.ContextType)
	}

	if !isUserTokenType(cfg.CloudContext.TokenType) {
		return "", fmt.Errorf("not authenticated via a user login flow (current: %s)", cfg.CloudContext.TokenType)
	}

	if cfg.CloudContext.ApiKey == "" {
		return "", fmt.Errorf("no access token available")
	}

	if cfg.CloudContext.RefreshToken == "" {
		return "", fmt.Errorf("no refresh token available - not a valid login session")
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

	if cfg.ContextType != config.ContextTypeCloud {
		return "", fmt.Errorf("not in cloud context")
	}

	if !isUserTokenType(cfg.CloudContext.TokenType) {
		return "", fmt.Errorf("not authenticated via a user login flow")
	}

	if cfg.CloudContext.ApiKey == "" || cfg.CloudContext.RefreshToken == "" {
		return "", fmt.Errorf("missing login tokens")
	}

	newAccessToken, newRefreshToken, err := refreshUserToken(context.Background(), cfg, cfg.SkipTLS || cfg.CloudContext.SkipTLS)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
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

// RefreshContextToken renews an expired user login token in place, so a command
// that talks to the Control Plane without going through GetClient - which does
// this for every other command - is not left using a token that expired.
//
// A stored API key has nothing to refresh, and neither has a login that never
// produced a refresh token, so both are left untouched and reported as no-ops.
func RefreshContextToken(cfg *config.Data, skipTLS bool) error {
	if !isUserTokenType(cfg.CloudContext.TokenType) {
		return nil
	}
	if cfg.CloudContext.ApiKey == "" || cfg.CloudContext.RefreshToken == "" {
		return nil
	}

	token, refreshToken, err := refreshUserToken(context.Background(), *cfg, skipTLS)
	if err != nil {
		return err
	}

	cfg.CloudContext.ApiKey = token
	cfg.CloudContext.RefreshToken = refreshToken

	return nil
}

// refreshUserToken dispatches the refresh path based on the stored TokenType.
// Returns a fresh (idToken, refreshToken); does not persist to config.
// The email-link path skips the network when the current idToken is still valid
// (verify-first), matching CheckAndRefreshToken's behavior for OIDC.
func refreshUserToken(ctx context.Context, cfg config.Data, skipTLS bool) (string, string, error) {
	switch cfg.CloudContext.TokenType {
	case config.TokenTypeEmailLink:
		return cloudlogin.RefreshEmailLinkToken(ctx, cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.RefreshToken, skipTLS)
	default:
		authURI := cfg.CloudContext.AuthUri
		if authURI == "" {
			authURI = fmt.Sprintf("%s/idp", cfg.CloudContext.ApiUri)
		}
		return cloudlogin.CheckAndRefreshToken(ctx, authURI, cfg.CloudContext.ApiKey, cfg.CloudContext.RefreshToken, skipTLS)
	}
}

// IsOAuthAuthenticated returns true only if strictly OAuth authenticated
func IsOAuthAuthenticated() bool {
	_, err := GetOAuthAccessToken()
	return err == nil
}
