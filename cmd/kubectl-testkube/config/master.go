package config

type Master struct {
	AgentToken     string `json:"agentToken"`
	IdToken        string `json:"idToken"`
	OrgId          string `json:"orgId"`
	EnvId          string `json:"envId"`
	Insecure       bool   `json:"insecure"`
	UiUrlPrefix    string `json:"uiUrlPrefix"`
	AgentUrlPrefix string `json:"agentUrlPrefix"`
	ApiUrlPrefix   string `json:"apiUrlPrefix"`
	RootDomain     string `json:"rootDomain"`

	URIs MasterURIs
}

type MasterURIs struct {
	UiPrefix   string `json:"uiPrefix"`
	ApiPrefix  string `json:"apiPrefix"`
	RootDomain string `json:"rootDomain"`
	Api        string `json:"api"`
	Agent      string `json:"agent"`
	Ui         string `json:"ui"`
	Auth       string `json:"auth"`
}
