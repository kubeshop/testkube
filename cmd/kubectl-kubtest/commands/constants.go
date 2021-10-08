package commands

import (
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

const (
	IngressApiServerName  string = "testkube-api-server"
	DashboardURI          string = "http://dashboard.testkube.io?apiEndpoint="
	DashboardPrefix       string = "testkube-dash"
	IngressControllerName string = "testkube-ing-ctrlr"
	CurrentApiVersion     string = apiclient.Version
)
