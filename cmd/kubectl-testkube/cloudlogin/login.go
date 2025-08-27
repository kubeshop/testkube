package cloudlogin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

const (
	clientID = "testkube-cloud-cli"
)

func getRedirectAddress(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", port)
}

func checkPortAvailable(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is not available: %w", port, err)
	}
	l.Close()
	return nil
}

type Tokens struct {
	IDToken      string
	RefreshToken string
}

func CloudLogin(ctx context.Context, providerURL, connectorID string, port int) (string, chan Tokens, error) {
	if err := checkPortAvailable(port); err != nil {
		return "", nil, err
	}

	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return "", nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: getRedirectAddress(port),
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

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
	srv := &http.Server{Addr: getRedirectAddress(port), Handler: mux}
	go func() {
		srv.ListenAndServe()
	}()

	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if connectorID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("connector_id", connectorID))
	}
	authURL := oauth2Config.AuthCodeURL("state", opts...)

	respCh := make(chan Tokens)

	go func() {
		defer srv.Close()

		code := <-ch

		token, err := oauth2Config.Exchange(ctx, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve token: %v\n", err)
			respCh <- Tokens{}
			return
		}

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
		// Attempt to refresh the token if verification fails
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

func CloudLoginSSO(ctx context.Context, apiBaseURL, authBaseURL, connectorID string, port int) (string, chan Tokens, error) {
	if err := checkPortAvailable(port); err != nil {
		return "", nil, err
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	redirectURI := getRedirectAddress(port)

	ch := make(chan string)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			ch <- code
			fmt.Fprintln(w, "<script>window.close()</script>")
			fmt.Fprintln(w, "Your testkube CLI is now successfully authenticated. Go back to the terminal to continue.")
		} else {
			fmt.Fprintln(w, "Authorization failed.")
		}
	})

	srv := &http.Server{Addr: getRedirectAddress(port), Handler: mux}
	go func() {
		srv.ListenAndServe()
	}()

	if !strings.HasSuffix(authBaseURL, "/") {
		authBaseURL += "/"
	}

	authURL, err := url.Parse(authBaseURL + "auth")
	if err != nil {
		return "", nil, fmt.Errorf("invalid auth URL: %w", err)
	}

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("connector_id", connectorID)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email offline_access")
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", "testkube-cli-state")
	authURL.RawQuery = params.Encode()

	respCh := make(chan Tokens)

	go func() {
		defer srv.Close()

		code := <-ch

		token, err := exchangeCodeForTokens(apiBaseURL, code, codeVerifier, port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to exchange code for tokens: %v\n", err)
			respCh <- Tokens{}
			return
		}

		respCh <- token
	}()

	return authURL.String(), respCh, nil
}

func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 96)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func exchangeCodeForTokens(apiBaseURL, code, codeVerifier string, port int) (Tokens, error) {
	if !strings.HasSuffix(apiBaseURL, "/") {
		apiBaseURL += "/"
	}

	tokenURL := apiBaseURL + "auth/login"

	redirectURI := getRedirectAddress(port)

	requestBody := map[string]string{
		"code":          code,
		"code_verifier": codeVerifier,
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(tokenURL, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Tokens{}, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return Tokens{}, fmt.Errorf("failed to decode token response: %w", err)
	}

	return Tokens{
		IDToken:      tokenResponse.IDToken,
		RefreshToken: tokenResponse.RefreshToken,
	}, nil
}
