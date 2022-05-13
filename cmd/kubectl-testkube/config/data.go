package config

import (
	"golang.org/x/oauth2"
)

type Data struct {
	AnalyticsEnabled bool       `json:"analyticsEnabled,omitempty"`
	Namespace        string     `json:"namespace,omitempty"`
	Initialized      bool       `json:"initialized,omitempty"`
	OAuth2Data       OAuth2Data `json:"oauth2_data"`
}

func (c *Data) EnableAnalytics() {
	c.AnalyticsEnabled = true

}

func (c *Data) DisableAnalytics() {
	c.AnalyticsEnabled = false
}

func (c *Data) SetNamespace(ns string) {
	c.Namespace = ns
}

func (c *Data) SetInitialized() {
	c.Initialized = true
}

type OAuth2Data struct {
	Enabled      bool            `json:"enabled,omitempty"`
	Endpoint     oauth2.Endpoint `json:"endpoint,omitempty"`
	Token        *oauth2.Token   `json:"token,omitempty"`
	CliendID     string          `json:"clientID,omitempty"`
	ClientSecret string          `json:"clientSecret,omitempty"`
	Scopes       []string        `json:"scopes,omitempty"`
}
