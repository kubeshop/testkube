package config

type Master struct {
	AgentToken     string
	IdToken        string
	OrgId          string
	EnvId          string
	Insecure       bool
	UiUrlPrefix    string
	AgentUrlPrefix string
	ApiUrlPrefix   string
	RootDomain     string

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
