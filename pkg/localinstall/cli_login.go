package localinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// The demo values bootstrap both.
const (
	DemoOrgID = "tkcorg_demo"
	DemoEnvID = "tkcenv_my-first-environment"
)

// The dashboard's client; the CLI's own one can't use passwords.
const dexClientID = "testkube-enterprise"

// Names must be unique per organization, hence the time.
const cliKeyName = "Testkube CLI (install local) "

var ErrCLILogin = errors.New("could not log the CLI in")

// A password login spares the user a browser round trip.
func CLIKey(ctx context.Context, ports Ports, email string) (string, error) {
	return cliKey(ctx, http.DefaultClient, fmt.Sprintf("http://localhost:%d", ports["login"]),
		fmt.Sprintf("http://localhost:%d", ports["api"]), email)
}

func cliKey(ctx context.Context, client *http.Client, dex, api, email string) (string, error) {
	secret, err := dexClientSecret()
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"password"}, "username": {email}, "password": {AdminPassword},
		"scope": {"openid email groups offline_access"}, "client_id": {dexClientID}, "client_secret": {secret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dex+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var login struct {
		IDToken string `json:"id_token"`
	}
	if err := call(client, req, &login); err != nil || login.IDToken == "" {
		return "", fmt.Errorf("%w: login: %v", ErrCLILogin, err)
	}

	scope := func(resource, id string) map[string]string {
		return map[string]string{"resource": resource, "identifier": id, "role": "admin"}
	}
	body, _ := json.Marshal(map[string]any{"name": cliKeyName + time.Now().Format("2006-01-02 15:04:05.000"),
		"scopes": []map[string]string{scope("org", DemoOrgID), scope("env", DemoEnvID)}})
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, api+"/organizations/"+DemoOrgID+"/tokens", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+login.IDToken)
	var key struct {
		Token string `json:"token"`
	}
	if err := call(client, req, &key); err != nil || key.Token == "" {
		return "", fmt.Errorf("%w: key: %v", ErrCLILogin, err)
	}
	return key.Token, nil
}

// Reruns keep a working key, so keys don't pile up.
func CLIKeyWorks(ctx context.Context, ports Ports, key string) bool {
	return cliKeyWorks(ctx, http.DefaultClient, fmt.Sprintf("http://localhost:%d", ports["api"]), key)
}

func cliKeyWorks(ctx context.Context, client *http.Client, api, key string) bool {
	if key == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api+"/organizations/"+DemoOrgID, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+key)
	var org map[string]any
	return call(client, req, &org) == nil
}

func call(client *http.Client, req *http.Request, out any) error {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s answered %d", req.URL.Path, resp.StatusCode)
	}
	return json.Unmarshal(data, out)
}

// Read from the values, so a chart bump can't drift.
func dexClientSecret() (string, error) {
	var demo map[string]any
	if err := yaml.Unmarshal(EnterpriseDemoValues, &demo); err != nil {
		return "", err
	}
	clients, _ := dig(demo, "dex", "configTemplate", "additionalStaticClients").([]any)
	for _, c := range clients {
		if m, _ := c.(map[string]any); m["id"] == dexClientID {
			if secret, _ := m["secret"].(string); secret != "" {
				return secret, nil
			}
		}
	}
	return "", errors.New("demo values have no dashboard login client")
}
