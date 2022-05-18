package oauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils"
	"golang.org/x/oauth2"

	"github.com/skratchdot/open-golang/open"
)

const (
	// localIP to open local website
	localIP = "127.0.0.1"
	// authTimeout is time to wait for authentication completed
	authTimeout = 120
	// oauthStateStringContextKey is a context key for oauth strategy
	oauthStateStringContextKey = 987
	// callbackPath is a path to callback handler
	callbackPath = "/oauth/callback"
	// redirectDelay is redirect delay
	redirectDelay = 15 * time.Second
	// shutdownTimeout is shutdown timeout
	shutdownTimeout = 5 * time.Second
	// randomLength is a length of a random string
	randomLength = 8
	// successPage is a page to show for success authentication
	successPage = `<h1>Success!</h1>
		<p>You are authenticated, you can now return to the program. This will auto-close</p>
		<script>window.onload=function(){setTimeout(this.close, 5000)}</script>`
)

// NewProvider returns new provider
func NewProvider(oauthConfig *oauth2.Config, port int) Provider {
	// add transport for self-signed certificate to context
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	oauthConfig.RedirectURL = fmt.Sprintf("http://%s:%d"+callbackPath, localIP, port)
	return Provider{
		oauthConfig: oauthConfig,
		client:      &http.Client{Transport: tr},
		port:        port,
	}
}

// Provider contains oauth provider config
type Provider struct {
	oauthConfig *oauth2.Config
	client      *http.Client
	port        int
}

type AuthorizedClient struct {
	Client *http.Client
	Token  *oauth2.Token
}

// AuthenticateUser starts the login process
func (p Provider) AuthenticateUser(values url.Values) (client *AuthorizedClient, err error) {
	oauthStateString, err := utils.NewRandomString(randomLength)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, p.client)
	ctx = context.WithValue(ctx, oauthStateStringContextKey, oauthStateString)

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

	p.startHTTPServer(ctx, clientChan, shutdownChan, cancelChan)

	ui.Info("You will be redirected to your browser for authentication or you can open the url below manually.")
	ui.Info(authURL)
	time.Sleep(redirectDelay)

	if err = open.Run(authURL); err != nil {
		return nil, err
	}

	// shutdown the server after timeout
	go func() {
		ui.Info(fmt.Sprintf("Authentication will be cancelled in %d seconds", authTimeout))
		time.Sleep(authTimeout * time.Second)

		shutdownChan <- struct{}{}
	}()

	// wait for an authenticated client or cancel authentication
	select {
	case client = <-clientChan:
		shutdownChan <- struct{}{}
	case <-cancelChan:
		err = fmt.Errorf("authentication timed out and was cancelled")
	}

	return client, err
}

func (p Provider) startHTTPServer(ctx context.Context, clientChan chan *AuthorizedClient,
	shutdownChan chan struct{}, cancelChan chan struct{}) {
	http.HandleFunc(callbackPath, p.CallbackHandler(ctx, clientChan))
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

		cancelChan <- struct{}{}
	}()

	// handle callback request
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			ui.ExitOnError("starting http server", err)
		}

		ui.Info("Server gracefully stopped")
	}()

	return
}

// CallbackHandler is oauth callback handler
func (p Provider) CallbackHandler(ctx context.Context, clientChan chan *AuthorizedClient) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestState, ok := ctx.Value(oauthStateStringContextKey).(string)
		if !ok {
			ui.Errf("unknown oauth state '%s'\n", ctx.Value(oauthStateStringContextKey))
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		responseState := r.FormValue("state")
		if responseState != requestState {
			ui.Errf("invalid oauth state, expected '%s', got '%s'\n", requestState, responseState)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		code := r.FormValue("code")
		token, err := p.oauthConfig.Exchange(ctx, code)
		if err != nil {
			ui.Errf("oauth exchange failed with '%s'\n", err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		fmt.Fprintf(w, successPage)

		clientChan <- &AuthorizedClient{
			Client: p.oauthConfig.Client(ctx, token),
			Token:  token,
		}
	}
}
