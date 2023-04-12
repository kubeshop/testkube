package commands

import (
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

const (
	ApiServerName      string = "testkube-api-server"
	ApiServerPort      int    = 8088
	ApiVersion         string = "v1"
	DashboardURI       string = "http://dashboard.testkube.io"
	CurrentApiVersion  string = apiclient.Version
	DashboardName      string = "testkube-dashboard"
	DashboardPort      int    = 8080
	DashboardLocalPort int    = 8080
)
