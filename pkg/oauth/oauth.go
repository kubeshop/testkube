package oauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils"
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
func NewProvider(oauthConfig *oauth2.Config) Provider {
	// add transport for self-signed certificate to context
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	oauthConfig.RedirectURL = fmt.Sprintf("http://%s:%d%s", localIP, localPort, callbackPath)
	return Provider{
		oauthConfig: oauthConfig,
		client:      &http.Client{Transport: tr},
		port:        localPort,
	}
}

// Provider contains oauth provider config
type Provider struct {
	oauthConfig *oauth2.Config
	client      *http.Client
	port        int
}

// AuthorizedClient is authorized client and token
type AuthorizedClient struct {
	Client *http.Client
	Token  *oauth2.Token
}

// ValidateToken validates token
func (p Provider) ValidateToken(token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := p.oauthConfig.TokenSource(context.Background(), token)
	return tokenSource.Token()
}

// AuthenticateUser starts the login process
func (p Provider) AuthenticateUser(values url.Values) (client *AuthorizedClient, err error) {
	oauthStateString, err := utils.NewRandomString(randomLength)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.WithValue(context.Background(), oauth2.HTTPClient, p.client),
		oauthStateStringContextKey, oauthStateString)

	authURL := p.oauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		return nil, err
	}

	params := parsedURL.Query()
	for key, value := range values {
		params[key] = value
	}

	parsedURL.RawQuery = params.Encode()
	authURL = parsedURL.String()

	clientChan := make(chan *AuthorizedClient)
	shutdownChan := make(chan struct{})
	cancelChan := make(chan struct{})

	p.startHTTPServer(ctx, clientChan, shutdownChan)

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
	shutdownChan chan struct{}) {
	http.HandleFunc(callbackPath, p.CallbackHandler(ctx, clientChan))
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
func (p Provider) CallbackHandler(ctx context.Context, clientChan chan *AuthorizedClient) func(w http.ResponseWriter, r *http.Request) {
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

		code := r.FormValue("code")
		token, err := p.oauthConfig.Exchange(ctx, code)
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
			Client: p.oauthConfig.Client(ctx, token),
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
