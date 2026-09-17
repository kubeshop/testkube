package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LicenseResponse struct {
	Valid   bool   `json:"valid,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	License struct {
		Expiry string `json:"expiry,omitempty"`
		Name   string `json:"name,omitempty"`
	} `json:"license,omitempty"`
}

type LicenseRequest struct {
	License string `json:"license"`
}

// EventRequest reports a product lifecycle event against a license.
type EventRequest struct {
	License string `json:"license"`
	Event   string `json:"event"`
}

const (
	EventCLIInstallStarted  = "cli_install_started"
	EventCLIInstallFinished = "cli_install_finished"
)

type Client struct {
	url string
}

const LicenseValidationURL = "https://license.testkube.io/validate"

const LicenseEventsURL = "https://license.testkube.io/events"

func NewClient() *Client {
	return &Client{url: LicenseValidationURL}
}

func (c *Client) WithURL(url string) *Client {
	c.url = url
	return c
}

func (c *Client) ValidateLicense(licenseRequest LicenseRequest) (*LicenseResponse, error) {
	reqBody, err := json.Marshal(licenseRequest)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var licenseResponse LicenseResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return &LicenseResponse{
			Valid:   false,
			Message: string(b),
		}, nil
	}

	err = json.Unmarshal(b, &licenseResponse)
	if err != nil {
		return nil, err
	}

	return &licenseResponse, nil
}

func (c *Client) ReportEvent(license, event string) error {
	reqBody, err := json.Marshal(EventRequest{License: license, Event: event})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, LicenseEventsURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("license event report failed with status %d", resp.StatusCode)
	}

	return nil
}
