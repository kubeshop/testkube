package toolkit_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestServiceFailureDetection_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	services := map[string]testworkflowsv1.ServiceSpec{
		"failing-service": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "alpine:latest",
					},
					Shell: common.Ptr(`exit 1`),
				},
			},
		},
	}

	err := executeServices(t, services, "test-group")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestServiceSuccess_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	services := map[string]testworkflowsv1.ServiceSpec{
		"healthy-service": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "alpine:latest",
					},
					Shell: common.Ptr(`sleep 300`),
				},
			},
		},
	}

	err := executeServices(t, services, "test-group")

	assert.NoError(t, err)
}

func executeServices(t *testing.T, services map[string]testworkflowsv1.ServiceSpec, groupRef string) error {
	data, err := json.Marshal(services)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(data)

	cfg, err := config.LoadConfigV2()
	require.NoError(t, err)

	return commands.RunServicesWithOptions(encoded, cfg, true, groupRef)
}
