package cloudlogin

import (
	"bytes"
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
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

const (
	clientID = "testkube-cloud-cli"
)

func getRedirectAddress(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", port)
}

func getServerAddress(port int) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func checkPortAvailable(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
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
	srv := &http.Server{Addr: getServerAddress(port), Handler: mux}
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
	provider, err := oidc.NewProvider(ctx, providerURL)
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

	srv := &http.Server{Addr: getServerAddress(port), Handler: mux}
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

func CloudLoginEmailLink(ctx context.Context, apiBaseURL, email string, port int) (chan Tokens, error) {
	if err := checkPortAvailable(port); err != nil {
		return nil, err
	}

	if !strings.HasSuffix(apiBaseURL, "/") {
		apiBaseURL += "/"
	}

	// The callback only needs the oobCode; the email is the one the caller
	// already passed to requestEmailLink, so trusting a query param is both
	// unnecessary and a gratuitous trust surface.
	oobCh := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		oobCode := r.URL.Query().Get("oobCode")
		if oobCode == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Authorization failed: missing oobCode.")
			// Don't signal on malformed callbacks — let the caller's timeout
			// surface the error rather than completing with empty tokens.
			return
		}
		fmt.Fprintln(w, "<script>window.close()</script>")
		fmt.Fprintln(w, "Your testkube CLI is now successfully authenticated. Go back to the terminal to continue.")
		select {
		case oobCh <- oobCode:
		default: // drop duplicate clicks
		}
	})
	srv := &http.Server{Addr: getServerAddress(port), Handler: mux}
	go func() {
		srv.ListenAndServe()
	}()

	if err := requestEmailLink(ctx, apiBaseURL, email, getRedirectAddress(port)); err != nil {
		srv.Close()
		return nil, err
	}

	respCh := make(chan Tokens, 1)

	go func() {
		defer srv.Close()

		var oobCode string
		select {
		case oobCode = <-oobCh:
		case <-ctx.Done():
			respCh <- Tokens{}
			return
		}

		tokens, err := exchangeOOBCodeForTokens(ctx, apiBaseURL, email, oobCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to exchange oobCode for tokens: %v\n", err)
			respCh <- Tokens{}
			return
		}

		respCh <- tokens
	}()

	return respCh, nil
}

// requestEmailLink asks the control plane to generate and email a sign-in link.
// The `state=redirect=<url>` convention is what HandleOutOfBandGenerateLink expects
// (see internal/auth/controller/out_of_band_flow.go).
func requestEmailLink(ctx context.Context, apiBaseURL, email, redirectURL string) error {
	form := url.Values{}
	form.Set("email", email)
	form.Set("state", "redirect="+redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"auth/link", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build email-link request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request email link: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email link request failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func exchangeOOBCodeForTokens(ctx context.Context, apiBaseURL, email, oobCode string) (Tokens, error) {
	u, err := url.Parse(apiBaseURL + "auth/login/link")
	if err != nil {
		return Tokens{}, fmt.Errorf("invalid auth/login/link URL: %w", err)
	}
	q := u.Query()
	q.Set("email", email)
	q.Set("oobCode", oobCode)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to build oobCode exchange request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to exchange oobCode: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Tokens{}, fmt.Errorf("oobCode exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return Tokens{}, fmt.Errorf("failed to decode oobCode exchange response: %w", err)
	}

	return Tokens{
		IDToken:      tokenResponse.IDToken,
		RefreshToken: tokenResponse.RefreshToken,
	}, nil
}

// RefreshEmailLinkToken returns a non-expired ID token, refreshing via the
// control plane only when the current idToken is within 60s of expiry. Matches
// the verify-first pattern used by CheckAndRefreshToken for OIDC so the common
// case (token still valid) doesn't hit the network.
func RefreshEmailLinkToken(ctx context.Context, apiBaseURL, idToken, refreshToken string) (string, string, error) {
	if idToken != "" && !jwtExpired(idToken, 60*time.Second) {
		return idToken, refreshToken, nil
	}

	if !strings.HasSuffix(apiBaseURL, "/") {
		apiBaseURL += "/"
	}

	body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"auth/callback?method=emailLink", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to refresh email-link token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("email-link refresh failed with status %d: %s", resp.StatusCode, string(b))
	}

	var refreshResp struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return "", "", fmt.Errorf("failed to decode refresh response: %w", err)
	}
	return refreshResp.IDToken, refreshResp.RefreshToken, nil
}

// decodeJWTPayload returns the raw JSON payload of a JWT without verifying its
// signature. Returns nil on any parse failure.
func decodeJWTPayload(token string) []byte {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	return payload
}

// jwtExpired reports whether a JWT's `exp` claim is already past (accounting for
// the provided skew). Returns true on any parse failure, so the caller errs on
// the side of refreshing rather than trusting an unparseable token.
func jwtExpired(token string, skew time.Duration) bool {
	payload := decodeJWTPayload(token)
	if payload == nil {
		return true
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return true
	}
	return time.Now().Add(skew).After(time.Unix(claims.Exp, 0))
}

// EmailFromIDToken returns the `email` claim from an ID token's payload without
// verifying the signature. Empty string on any parse failure.
func EmailFromIDToken(token string) string {
	payload := decodeJWTPayload(token)
	if payload == nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Email
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, tokenURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
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
