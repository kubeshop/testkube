package testworkflowexecutor

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func createStorageMachine() expressions.Machine {
	return expressions.NewMachine().
		RegisterStringMap("internal", map[string]string{
			"storage.url":        os.Getenv("STORAGE_ENDPOINT"),
			"storage.accessKey":  os.Getenv("STORAGE_ACCESSKEYID"),
			"storage.secretKey":  os.Getenv("STORAGE_SECRETACCESSKEY"),
			"storage.region":     os.Getenv("STORAGE_REGION"),
			"storage.bucket":     os.Getenv("STORAGE_BUCKET"),
			"storage.token":      os.Getenv("STORAGE_TOKEN"),
			"storage.ssl":        common.GetOr(os.Getenv("STORAGE_SSL"), "false"),
			"storage.skipVerify": common.GetOr(os.Getenv("STORAGE_SKIP_VERIFY"), "false"),
			"storage.certFile":   os.Getenv("STORAGE_CERT_FILE"),
			"storage.keyFile":    os.Getenv("STORAGE_KEY_FILE"),
			"storage.caFile":     os.Getenv("STORAGE_CA_FILE"),
		})
}

func createCloudMachine() expressions.Machine {
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	cloudOrgId := common.GetOr(os.Getenv("TESTKUBE_PRO_ORG_ID"), os.Getenv("TESTKUBE_CLOUD_ORG_ID"))
	cloudEnvId := common.GetOr(os.Getenv("TESTKUBE_PRO_ENV_ID"), os.Getenv("TESTKUBE_CLOUD_ENV_ID"))
	cloudUiUrl := common.GetOr(os.Getenv("TESTKUBE_PRO_UI_URL"), os.Getenv("TESTKUBE_CLOUD_UI_URL"))
	dashboardUrl := env.Config().System.DashboardUrl
	if cloudApiKey != "" {
		dashboardUrl = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard", cloudUiUrl, cloudOrgId, cloudEnvId)
	}
	return expressions.NewMachine().
		RegisterStringMap("internal", map[string]string{
			"cloud.enabled":         strconv.FormatBool(os.Getenv("TESTKUBE_PRO_API_KEY") != "" || os.Getenv("TESTKUBE_CLOUD_API_KEY") != ""),
			"cloud.api.key":         cloudApiKey,
			"cloud.api.tlsInsecure": common.GetOr(os.Getenv("TESTKUBE_PRO_TLS_INSECURE"), os.Getenv("TESTKUBE_CLOUD_TLS_INSECURE"), "false"),
			"cloud.api.skipVerify":  common.GetOr(os.Getenv("TESTKUBE_PRO_SKIP_VERIFY"), os.Getenv("TESTKUBE_CLOUD_SKIP_VERIFY"), "false"),
			"cloud.api.url":         common.GetOr(os.Getenv("TESTKUBE_PRO_URL"), os.Getenv("TESTKUBE_CLOUD_URL")),
			"cloud.ui.url":          cloudUiUrl,
			"cloud.api.orgId":       cloudOrgId,
			"cloud.api.envId":       cloudEnvId,

			"dashboard.url": os.Getenv("TESTKUBE_DASHBOARD_URI"),
		}).
		Register("organization", map[string]string{
			"id": cloudOrgId,
		}).
		Register("environment", map[string]string{
			"id": cloudEnvId,
		}).
		Register("dashboard", map[string]string{
			"url": dashboardUrl,
		})
}

func createWorkflowMachine(workflow testworkflowsv1.TestWorkflow) expressions.Machine {
	escapedLabels := make(map[string]string)
	for key, value := range workflow.Labels {
		escapedLabels[expressions.EscapeLabelKeyForVarName(key)] = value
	}
	return expressions.NewMachine().
		Register("workflow", map[string]interface{}{
			"name":   workflow.Name,
			"labels": workflow.Labels,
		}).
		RegisterStringMap("labels", escapedLabels)
}

func createResourceMachine(resourceId, rootResourceId, fsPrefix string) expressions.Machine {
	return expressions.NewMachine().Register("resource", map[string]string{
		"id":       resourceId,
		"root":     rootResourceId,
		"fsPrefix": fsPrefix,
	})
}

func validateWorkflow(ctx context.Context, processor testworkflowprocessor.Processor, workflow testworkflowsv1.TestWorkflow, disableWebhooks bool, machines ...expressions.Machine) error {
	machines = append([]expressions.Machine{
		expressions.NewMachine().Register("execution", map[string]interface{}{
			"id":              "507f191e810c19729de860ea",
			"name":            "<mock_name>",
			"number":          "1",
			"scheduledAt":     time.Now().UTC().Format(constants.RFC3339Millis),
			"disableWebhooks": disableWebhooks,
			"tags":            "", // FIXME?
		}),
	}, machines...)
	_, err := processor.Bundle(ctx, workflow.DeepCopy(), testworkflowprocessor.BundleOptions{}, machines...)
	return err
}
