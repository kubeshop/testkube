package license

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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

type Client struct {
	url string
}

const LicenseValidationURL = "https://license.testkube.io/validate"

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

	resp, err := http.Post(c.url, "application/json", bytes.NewBuffer(reqBody))
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
