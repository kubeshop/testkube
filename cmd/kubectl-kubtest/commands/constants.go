package commands

import (
	apiclient "github.com/kubeshop/kubtest/pkg/api/v1/client"
)

const (
	IngressApiServerName  string = "kubtest-api-server"
	DashboardURI          string = "http://dashboard.kubtest.io?apiEndpoint="
	DashboardPrefix       string = "kubtest-dash"
	IngressControllerName string = "kubtest-ing-ctrlr"
	CurrentApiVersion     string = apiclient.Version
)
