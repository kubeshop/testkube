package commands

import (
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

const (
	ApiServerName     string = "testkube-api-server"
	ApiServerPort     int    = 8088
	ApiVersion        string = "v1"
	DashboardURI      string = "http://demo.testkube.io/apiEndpoint?apiEndpoint="
	CurrentApiVersion string = apiclient.Version
)
