package common

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

const (
	defaultAgentPort   = 443
	defaultLogsPort    = 443
	defaultAgentPrefix = "agent"
	defaultUiPrefix    = "app"
	defaultLogsPrefix  = "logs"
	defaultApiPrefix   = "api"
	defaultRootDomain  = "testkube.io"
)

func NewMasterUris(apiPrefix, uiPrefix, agentPrefix, logsPrefix, agentURI, logsURI, rootDomain string, insecure bool) config.MasterURIs {
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
	if logsPrefix == "" {
		logsPrefix = defaultLogsPrefix
	}
	if rootDomain == "" {
		rootDomain = defaultRootDomain
	}
	if agentURI == "" {
		agentURI = fmt.Sprintf("%s.%s:%d", agentPrefix, rootDomain, defaultAgentPort)
	}
	if logsURI == "" {
		logsURI = fmt.Sprintf("%s.%s:%d", logsPrefix, rootDomain, defaultLogsPort)
	}

	return config.MasterURIs{
		ApiPrefix:  apiPrefix,
		UiPrefix:   uiPrefix,
		RootDomain: rootDomain,
		Api:        fmt.Sprintf("%s://%s.%s", protocol, apiPrefix, rootDomain),
		Agent:      agentURI,
		Logs:       logsURI,
		Ui:         fmt.Sprintf("%s://%s.%s", protocol, uiPrefix, rootDomain),
		Auth:       fmt.Sprintf("%s://%s.%s/idp", protocol, apiPrefix, rootDomain),
	}
}
