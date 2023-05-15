package common

import "fmt"

const defaultAgentPort = 443

func NewCloudUris(rootDomain string) CloudUris {
	return CloudUris{
		RootDomain: rootDomain,
		Api:        fmt.Sprintf("https://api.%s", rootDomain),
		Agent:      fmt.Sprintf("agent.%s:%d", rootDomain, defaultAgentPort),
		Ui:         fmt.Sprintf("https://cloud.%s", rootDomain),
		Auth:       fmt.Sprintf("https://api.%s/idp", rootDomain),
	}
}

type CloudUris struct {
	RootDomain string `json:"rootDomain"`
	Api        string `json:"api"`
	Agent      string `json:"agent"`
	Ui         string `json:"ui"`
	Auth       string `json:"auth"`
}
