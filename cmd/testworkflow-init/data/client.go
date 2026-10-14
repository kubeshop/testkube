package data

import (
	"context"
	"strings"
	"sync"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/internal/config"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/log"
)

// storageSkipVerify decides whether object storage certificates are verified. Storage in
// a self-hosted deployment commonly presents a certificate from a private CA that this
// image has no reason to trust, and the control plane hands out presigned URLs for that
// host. Kept as one constant so that uploading and downloading cannot disagree about it.
const storageSkipVerify = true

var (
	cloudMu     sync.Mutex
	cloudClient controlplaneclient.Client
)

func CloudClient() controlplaneclient.Client {
	cloudMu.Lock()
	defer cloudMu.Unlock()

	if cloudClient == nil {
		cfg := GetState().InternalConfig
		conn := cfg.Worker.Connection
		logger := log.NewSilent()
		grpcConn, err := agentclient.NewVeryInsecureGRPCClientDoNotUseThisClientUnlessYouAreReallySureYouKnowWhatYouAreDoing(context.Background(), conn.TlsInsecure, conn.Url, logger)
		if err != nil {
			output.ExitErrorf(constants.CodeInternal, "failed to connect with the Control Plane: %s", err.Error())
		}
		cloudClient = controlplaneclient.New(cloud.NewTestKubeCloudAPIClient(grpcConn), config.ProContext{
			APIKey:      conn.ApiKey,
			URL:         conn.Url,
			TLSInsecure: conn.TlsInsecure,
			SkipVerify:  conn.SkipVerify,
			EnvID:       cfg.Execution.EnvironmentId,
			OrgID:       cfg.Execution.OrganizationId,
			Agent: config.ProContextAgent{
				ID:   cfg.Worker.Connection.AgentID,
				Name: cfg.Worker.Connection.AgentID,
				Environments: []config.ProContextAgentEnvironment{
					{
						ID:   cfg.Execution.EnvironmentId,
						Slug: cfg.Execution.EnvironmentId,
						Name: cfg.Execution.EnvironmentId,
					},
				},
			},
		}, controlplaneclient.ClientOptions{
			StorageSkipVerify:  storageSkipVerify,
			ExecutionID:        cfg.Execution.Id,
			WorkflowName:       cfg.Workflow.Name,
			ParentExecutionIDs: strings.Split(cfg.Execution.ParentIds, "/"),
		}, log.DefaultLogger)
	}
	return cloudClient
}

func Credentials() credentials.CredentialRepository {
	cfg := GetState().InternalConfig
	return credentials.NewCredentialRepository(CloudClient, cfg.Execution.EnvironmentId, cfg.Execution.Id)
}
