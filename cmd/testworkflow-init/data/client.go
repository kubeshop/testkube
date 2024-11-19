package data

import (
	"context"
	"sync"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/log"
)

var (
	cloudMu     sync.Mutex
	cloudClient cloud.TestKubeCloudAPIClient
)

func CloudClient() cloud.TestKubeCloudAPIClient {
	cloudMu.Lock()
	defer cloudMu.Unlock()

	if cloudClient == nil {
		cfg := GetState().InternalConfig.Worker.Connection
		logger := log.NewSilent()
		grpcConn, err := client.NewGRPCConnection(context.Background(), cfg.TlsInsecure, cfg.SkipVerify, cfg.Url, "", "", "", logger)
		if err != nil {
			output.ExitErrorf(constants.CodeInternal, "failed to connect with the Control Plane: %s", err.Error())
		}
		cloudClient = cloud.NewTestKubeCloudAPIClient(grpcConn)
	}
	return cloudClient
}

func Credentials() credentials.CredentialRepository {
	return credentials.NewCredentialRepository(CloudClient())
}
