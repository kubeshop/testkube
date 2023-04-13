package commands

import (
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

const (
	ApiVersion         string = "v1"
	DashboardURI       string = "http://dashboard.testkube.io"
	CurrentApiVersion  string = apiclient.Version
	DashboardLocalPort int    = 8080
)
