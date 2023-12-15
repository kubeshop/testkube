package common

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

const (
	defaultAgentPort   = 443
	defaultAgentPrefix = "agent"
	defaultUiPrefix    = "ui"
	defaultApiPrefix   = "api"
	defaultRootDomain  = "testkube.io"
)

func NewMasterUris(apiPrefix, uiPrefix, agentPrefix, rootDomain string, insecure bool) config.MasterURIs {
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

	return config.MasterURIs{
		ApiPrefix:  apiPrefix,
		RootDomain: rootDomain,
		Api:        fmt.Sprintf("%s://%s.%s", protocol, apiPrefix, rootDomain),
		Agent:      fmt.Sprintf("%s.%s:%d", agentPrefix, rootDomain, defaultAgentPort),
		Ui:         fmt.Sprintf("%s://%s.%s", protocol, uiPrefix, rootDomain),
		Auth:       fmt.Sprintf("%s://%s.%s/idp", protocol, apiPrefix, rootDomain),
	}
}
