package config

type Master struct {
	AgentToken     string `json:"agentToken,omitempty"`
	IdToken        string `json:"idToken,omitempty"`
	OrgId          string `json:"orgId,omitempty"`
	EnvId          string `json:"envId,omitempty"`
	Insecure       bool   `json:"insecure,omitempty"`
	UiUrlPrefix    string `json:"uiUrlPrefix,omitempty"`
	AgentUrlPrefix string `json:"agentUrlPrefix,omitempty"`
	ApiUrlPrefix   string `json:"apiUrlPrefix,omitempty"`
	RootDomain     string `json:"rootDomain,omitempty"`

	URIs MasterURIs `json:"uris,omitempty"`
}

type MasterURIs struct {
	UiPrefix   string `json:"uiPrefix,omitempty"`
	ApiPrefix  string `json:"apiPrefix,omitempty"`
	RootDomain string `json:"rootDomain,omitempty"`
	Api        string `json:"api,omitempty"`
	Agent      string `json:"agent,omitempty"`
	Ui         string `json:"ui,omitempty"`
	Auth       string `json:"auth,omitempty"`
}
