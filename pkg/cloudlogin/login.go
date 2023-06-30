package cloudlogin

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

const (
	clientID    = "testkube-cloud-cli"
	redirectURL = "http://127.0.0.1:8090/callback"
)

type Tokens struct {
	IDToken      string
	RefreshToken string
}

func CloudLogin(ctx context.Context, providerURL, connectorID string) (string, chan Tokens, error) {
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return "", nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: redirectURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

	// Start a local server to handle the callback from the OIDC provider.
	ch := make(chan string)
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			ch <- code
			fmt.Fprintln(w, "Your testkube CLI is now succesfully authenticated. Go back to the terminal to continue.")
		} else {
			fmt.Fprintln(w, "Authorization failed.")
		}
	})
	go http.ListenAndServe(":8090", nil)

	// Redirect the user to the OIDC provider's login page.
	authURL := oauth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("connector_id", connectorID))

	respCh := make(chan Tokens)

	go func() {
		// Wait for the user to authorize the client and retrieve the authorization code.
		code := <-ch

		// Exchange the authorization code for an access token and ID token.
		token, err := oauth2Config.Exchange(ctx, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve token: %v\n", err)
			respCh <- Tokens{}
			return
		}

		// Initialize the OIDC verifier with the provider's public keys.
		verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

		_, err = verifier.Verify(ctx, token.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to verify ID token: %v\n", err)
			respCh <- Tokens{}
			return
		}

		respCh <- Tokens{
			IDToken:      token.AccessToken,
			RefreshToken: token.RefreshToken,
		}
	}()

	return authURL, respCh, nil
}

func CheckAndRefreshToken(ctx context.Context, providerURL, rawIDToken, refreshToken string) (string, string, error) {
	provider, err := oidc.NewProvider(context.Background(), providerURL)
	if err != nil {
		return "", "", err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	_, err = verifier.Verify(ctx, rawIDToken)
	if err != nil {
		// Token is expired. Refresh it.
		oauth2Config := oauth2.Config{
			ClientID: clientID,
			Endpoint: provider.Endpoint(),
			Scopes:   []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
		}

		tokenSource := oauth2Config.TokenSource(ctx, &oauth2.Token{
			RefreshToken: refreshToken,
		})
		token, err := tokenSource.Token()
		if err != nil {
			return "", "", err
		}

		return token.AccessToken, token.RefreshToken, nil
	}
	return rawIDToken, refreshToken, nil
}
