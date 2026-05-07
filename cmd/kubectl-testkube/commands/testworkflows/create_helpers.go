package testworkflows

import (
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	common2 "github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

// CreateOrUpdateFromBytes validates the TestWorkflow YAML and either creates
// or updates the resource using the client bound to the command. It is
// shared between `testkube create testworkflow` and
// `testkube marketplace install` so both entry points share the same
// validation and upsert semantics.
//
// The resolved workflow name (after --name override, if any) is returned so
// callers can perform follow-up operations such as triggering a run.
// When dryRun is true the function exits after successful schema validation
// (matching the existing CLI behavior) and does not return.
func CreateOrUpdateFromBytes(cmd *cobra.Command, raw []byte, overrideName string, update, dryRun bool) string {
	namespace := cmd.Flag("namespace").Value.String()

	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	err = client.ValidateTestWorkflow(raw)
	ui.ExitOnError("error validating test workflow against crd schema", err)
	if dryRun {
		ui.SuccessAndExit("TestWorkflow specification is valid")
	}

	obj := new(testworkflowsv1.TestWorkflow)
	err = common2.DeserializeCRD(obj, raw)
	ui.ExitOnError("deserializing input", err)

	common2.AppendTypeMeta("TestWorkflow", testworkflowsv1.GroupVersion, obj)
	obj.Namespace = namespace
	if overrideName != "" {
		obj.Name = overrideName
	}

	workflow, err := client.GetTestWorkflow(obj.Name)
	if err != nil {
		if update {
			ui.WarnOnError("getting test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
		} else {
			ui.Debug("getting test workflow "+obj.Name+" in namespace "+obj.Namespace, err.Error())
		}
	}

	if workflow.Name != "" {
		if !update {
			ui.Failf("Test workflow with name '%s' already exists in namespace %s, use --update flag for upsert", obj.Name, namespace)
		}
		_, err = client.UpdateTestWorkflow(testworkflows.MapTestWorkflowKubeToAPI(*obj))
		ui.ExitOnError("updating test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
		ui.Success("Test workflow updated", namespace, "/", obj.Name)
		return obj.Name
	}

	_, err = client.CreateTestWorkflow(testworkflows.MapTestWorkflowKubeToAPI(*obj))
	ui.ExitOnError("creating test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
	ui.Success("Test workflow created", namespace, "/", obj.Name)
	return obj.Name
}
