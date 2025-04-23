package cloudlogin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	clientID    = "testkube-cloud-cli"
	redirectURL = "http://127.0.0.1:%d/callback"
)

type Tokens struct {
	IDToken      string
	RefreshToken string
}

func CloudLogin(ctx context.Context, providerURL, connectorID string, port int) (string, chan Tokens, error) {
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return "", nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: fmt.Sprintf(redirectURL, port),
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

	// Start a local server to handle the callback from the OIDC provider.
	ch := make(chan string)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			ch <- code
			fmt.Fprintln(w, "<script>window.close()</script>")
			fmt.Fprintln(w, "Your testkube CLI is now succesfully authenticated. Go back to the terminal to continue.")
		} else {
			fmt.Fprintln(w, "Authorization failed.")
		}
	})
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			if strings.Contains(err.Error(), "address already in use") {
				ui.Fail(errors.Wrap(err, "failed to start callback server, you may try again with `--callback-port 38090`"))
			} else {
				ui.Fail(errors.Wrap(err, "failed to start callback server"))
			}
		}
	}()

	// Redirect the user to the OIDC provider's login page.
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if connectorID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("connector_id", connectorID))
	}
	authURL := oauth2Config.AuthCodeURL("state", opts...)

	respCh := make(chan Tokens)

	go func() {
		// Close the callback server
		defer srv.Close()

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
