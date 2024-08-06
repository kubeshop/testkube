package config

import "github.com/kubeshop/testkube/pkg/featureflags"

type Master struct {
	AgentToken     string                    `json:"agentToken,omitempty"`
	IdToken        string                    `json:"idToken,omitempty"`
	OrgId          string                    `json:"orgId,omitempty"`
	EnvId          string                    `json:"envId,omitempty"`
	RunnerId       string                    `json:"runnerId,omitempty"`
	Insecure       bool                      `json:"insecure,omitempty"`
	UiUrlPrefix    string                    `json:"uiUrlPrefix,omitempty"`
	AgentUrlPrefix string                    `json:"agentUrlPrefix,omitempty"`
	LogsUrlPrefix  string                    `json:"logsUrlPrefix,omitempty"`
	ApiUrlPrefix   string                    `json:"apiUrlPrefix,omitempty"`
	RootDomain     string                    `json:"rootDomain,omitempty"`
	Features       featureflags.FeatureFlags `json:"features,omitempty"`

	URIs MasterURIs `json:"uris,omitempty"`
}

type MasterURIs struct {
	UiPrefix   string `json:"uiPrefix,omitempty"`
	ApiPrefix  string `json:"apiPrefix,omitempty"`
	RootDomain string `json:"rootDomain,omitempty"`
	Api        string `json:"api,omitempty"`
	Agent      string `json:"agent,omitempty"`
	Logs       string `json:"logs,omitempty"`
	Ui         string `json:"ui,omitempty"`
	Auth       string `json:"auth,omitempty"`
}

// WithApi sets whole api URI
func (m *MasterURIs) WithApiURI(uri string) *MasterURIs {
	m.Api = uri
	return m
}

// WithAgent sets whole agent URI
func (m *MasterURIs) WithAgentURI(uri string) *MasterURIs {
	m.Agent = uri
	return m
}

// WithLogs sets whole logs URI
func (m *MasterURIs) WithLogsURI(uri string) *MasterURIs {
	m.Logs = uri
	return m
}

// WithUi sets whole ui URI
func (m *MasterURIs) WithUiURI(uri string) *MasterURIs {
	m.Ui = uri
	return m
}
