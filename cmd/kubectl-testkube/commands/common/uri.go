package common

import (
	"fmt"
)

const (
	defaultAgentPort   = 443
	defaultAgentPrefix = "agent"
	defaultUiPrefix    = "ui"
	defaultApiPrefix   = "api"
	defaultRootDomain  = "testkube.io"
)

func NewCloudUris(apiPrefix, uiPrefix, agentPrefix, rootDomain string, insecure bool) CloudUris {
	protocol := "https"
	if insecure {
		protocol = "http"
	}
	if apiPrefix == "" {
		apiPrefix = defaultApiPrefix
	}
	if uiPrefix == "" {
		uiPrefix = defaultUiPrefix
	}
	if agentPrefix == "" {
		agentPrefix = defaultAgentPrefix
	}
	if rootDomain == "" {
		rootDomain = "testkube.io"
	}

	return CloudUris{
		ApiPrefix:  apiPrefix,
		RootDomain: rootDomain,
		Api:        fmt.Sprintf("%s://%s.%s", protocol, apiPrefix, rootDomain),
		Agent:      fmt.Sprintf("%s.%s:%d", agentPrefix, rootDomain, defaultAgentPort),
		Ui:         fmt.Sprintf("%s://%s.%s", protocol, uiPrefix, rootDomain),
		Auth:       fmt.Sprintf("%s://%s.%s/idp", protocol, apiPrefix, rootDomain),
	}
}

type CloudUris struct {
	UiPrefix   string `json:"uiPrefix"`
	ApiPrefix  string `json:"apiPrefix"`
	RootDomain string `json:"rootDomain"`
	Api        string `json:"api"`
	Agent      string `json:"agent"`
	Ui         string `json:"ui"`
	Auth       string `json:"auth"`
}
