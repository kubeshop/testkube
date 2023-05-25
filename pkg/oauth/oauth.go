package oauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/kubeshop/testkube/pkg/ui"
)

// key is context key
type key int

const (
	// localIP is ip uo open local website
	localIP = "127.0.0.1"
	// localPort is port to open local website
	localPort = 13254
	// authTimeout is time to wait for authentication completed
	authTimeout = 60
	// oauthStateStringContextKey is a context key for oauth strategy
	oauthStateStringContextKey key = 987
	// callbackPath is a path to callback handler
	callbackPath = "/oauth/callback"
	// errorPath is a path to error handler
	errorPath = "/oauth/error"
	// redirectDelay is redirect delay
	redirectDelay = 10 * time.Second
	// shutdownTimeout is shutdown timeout
	shutdownTimeout = 5 * time.Second
	// randomLength is a length of a random string
	randomLength = 8
	// successPage is a page to show for success authentication
	successPage = `<html><body><h2>Success!</h2>
		<p>You are authenticated, you can now return to the program.</p></body></html>`
	// errorPage is a page to show for failed authentication
	errorPage = `<html><body><h2>Error!</h2>
		<p>Authentication was failed, please check the program logs.</p></body</html>`
	// AuthorizationPrefix is authorization prefix
	AuthorizationPrefix = "Bearer"
)

// NewProvider returns new provider
func NewProvider(clientID, clientSecret string, scopes []string) Provider {
	// add transport for self-signed certificate to context
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}

	client := &http.Client{Transport: tr}
	provider := Provider{
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       scopes,
		client:       client,
		port:         localPort,
		validators:   map[ProviderType]Validator{},
	}

	provider.AddValidator(GithubProviderType, NewGithubValidator(client, clientID, clientSecret, scopes))
	return provider
}

// Provider contains oauth provider config
type Provider struct {
	clientID     string
	clientSecret string
	scopes       []string
	client       *http.Client
	port         int
	validators   map[ProviderType]Validator
}

// AuthorizedClient is authorized client and token
type AuthorizedClient struct {
	Client *http.Client
	Token  *oauth2.Token
}

func (p Provider) getOAuthConfig(providerType ProviderType) (*oauth2.Config, error) {
	validator, err := p.GetValidator(providerType)
	if err != nil {
		return nil, err
	}

	redirectURL := fmt.Sprintf("http://%s:%d%s", localIP, localPort, callbackPath)
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Endpoint:     validator.GetEndpoint(),
		RedirectURL:  redirectURL,
		Scopes:       p.scopes,
	}, nil
}

// AddValidator adds validator
func (p Provider) AddValidator(providerType ProviderType, validator Validator) {
	p.validators[providerType] = validator
}

// GetValidator returns validator
func (p Provider) GetValidator(providerType ProviderType) (Validator, error) {
	validator, ok := p.validators[providerType]
	if !ok {
		return nil, fmt.Errorf("unknown oauth provider %s", providerType)
	}

	return validator, nil
}

// ValidateToken validates token
func (p Provider) ValidateToken(providerType ProviderType, token *oauth2.Token) (*oauth2.Token, error) {
	config, err := p.getOAuthConfig(providerType)
	if err != nil {
		return nil, err
	}

	tokenSource := config.TokenSource(context.Background(), token)
	return tokenSource.Token()
}

// ValidateAccessToken validates access token
func (p Provider) ValidateAccessToken(providerType ProviderType, accessToken string) error {
	validator, err := p.GetValidator(providerType)
	if err != nil {
		return err
	}

	return validator.Validate(accessToken)
}

// AuthenticateUser starts the login process
func (p Provider) AuthenticateUser(providerType ProviderType) (client *AuthorizedClient, err error) {
	oauthStateString := rand.String(randomLength)
	ctx := context.WithValue(context.WithValue(context.Background(), oauth2.HTTPClient, p.client),
		oauthStateStringContextKey, oauthStateString)

	config, err := p.getOAuthConfig(providerType)
	if err != nil {
		return nil, err
	}

	authURL := config.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)

	clientChan := make(chan *AuthorizedClient)
	shutdownChan := make(chan struct{})
	cancelChan := make(chan struct{})

	p.startHTTPServer(ctx, clientChan, shutdownChan, providerType)

	ui.Info("You will be redirected to your browser for authentication or you can open the url below manually")
	ui.Info(authURL)

	time.Sleep(redirectDelay)

	if err = open.Run(authURL); err != nil {
		return nil, err
	}

	// shutdown the server after timeout
	go func() {
		ui.Info(fmt.Sprintf("Authentication will be cancelled in %d seconds", authTimeout))
		time.Sleep(authTimeout * time.Second)

		cancelChan <- struct{}{}
	}()

	// wait for an authenticated client or cancel authentication
	select {
	case client = <-clientChan:
	case <-cancelChan:
		err = fmt.Errorf("authentication timed out and was cancelled")
	}

	shutdownChan <- struct{}{}
	return client, err
}

// startHTTPServer starts http server
func (p Provider) startHTTPServer(ctx context.Context, clientChan chan *AuthorizedClient,
	shutdownChan chan struct{}, providerType ProviderType) {
	http.HandleFunc(callbackPath, p.CallbackHandler(ctx, clientChan, providerType))
	http.HandleFunc(errorPath, p.ErrorHandler())
	srv := &http.Server{Addr: ":" + strconv.Itoa(p.port)}

	// handle server shutdown signal
	go func() {
		<-shutdownChan

		ui.Info("Shutting down server...")

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(shutdownTimeout))
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			ui.Errf("stopping http server: %v", err)
		}
	}()

	// handle callback request
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			ui.ExitOnError("starting http server", err)
		}

		ui.Success("Server gracefully stopped")
	}()
}

// CallbackHandler is oauth callback handler
func (p Provider) CallbackHandler(ctx context.Context, clientChan chan *AuthorizedClient,
	providerType ProviderType) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestState, ok := ctx.Value(oauthStateStringContextKey).(string)
		if !ok {
			ui.Errf("unknown oauth state: %v", ctx.Value(oauthStateStringContextKey))
			http.Redirect(w, r, errorPath, http.StatusTemporaryRedirect)
			return
		}

		responseState := r.FormValue("state")
		if responseState != requestState {
			ui.Errf("invalid oauth state, expected %s, got %s", requestState, responseState)
			http.Redirect(w, r, errorPath, http.StatusTemporaryRedirect)
			return
		}

		config, err := p.getOAuthConfig(providerType)
		if err != nil {
			ui.Errf("getting oauth config: %v", err)
			http.Redirect(w, r, errorPath, http.StatusTemporaryRedirect)
			return
		}

		code := r.FormValue("code")
		token, err := config.Exchange(ctx, code)
		if err != nil {
			ui.Errf("exchanging oauth code: %v", err)
			http.Redirect(w, r, errorPath, http.StatusTemporaryRedirect)
			return
		}

		if _, err = fmt.Fprint(w, successPage); err != nil {
			ui.Errf("showing success page: %v", err)
			http.Redirect(w, r, errorPath, http.StatusTemporaryRedirect)
			return
		}

		clientChan <- &AuthorizedClient{
			Client: config.Client(ctx, token),
			Token:  token,
		}
	}
}

// ErrorHandler is oauth error handler
func (p Provider) ErrorHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, errorPage); err != nil {
			ui.Errf("showing success page: %v", err)
			return
		}
	}
}
