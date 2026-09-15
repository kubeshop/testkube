package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/common"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallAgentCommand() *cobra.Command {
	var namespace string

	cmd := &cobra.Command{
		Use:  "agent <name>",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check for deprecated --type flag usage
			if cmd.Flags().Changed("type") {
				ui.Warn("⚠️  The --type/-t flag is deprecated.")
				ui.Info("Please use capability flags instead:")
				ui.Info("  --execution : Enable execution capability")
				ui.Info("  --listener  : Enable listener capability")
				ui.Info("  --gitops    : Enable GitOps capability")
				ui.Info("  --webhooks  : Enable webhooks capability")
				ui.NL()
				return
			}

			UiInstallAgent(cmd, strings.Join(args, ""), []string{"testkube.io/source=cloud"})
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")
	common2.PopulateRunnerFlags(cmd)
	return cmd
}

func NewInstallCRDCommand() *cobra.Command {
	var (
		namespace   string
		releaseName string
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:  "crd",
		Args: cobra.MaximumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			UiInstallCRD(cmd, namespace, releaseName, dryRun)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the Helm Chart")
	cmd.Flags().StringVarP(&releaseName, "release-name", "r", "testkube-crd", "Helm Chart release name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "display helm commands only")

	return cmd
}

func UiInstallCRD(cmd *cobra.Command, namespace string, releaseName string, dryRun bool) {
	spinner := ui.NewSpinner("Fetching current CRDs")
	currentNamespace, currentReleaseName, installed, err := GetCRDInstallation()
	if err != nil {
		spinner.Fail()
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrResourceLookupFailed,
			"Error getting the installed CRDs",
			common2.ClusterLookupHint,
			err,
		))
	}

	if installed && currentReleaseName == "" {
		spinner.Fail()
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrInvalidInstallConfig,
			"The CRDs are not managed by the Testkube Helm Chart",
			"Delete the Testkube CRDs by hand, so that this command can install them with Helm",
			fmt.Errorf("the CRDs are installed, but they carry no Helm release annotation"),
		))
	}

	if installed {
		spinner.Success(fmt.Sprintf("The CRDs are installed already in '%s' namespace", currentNamespace))
		namespace = currentNamespace
		releaseName = currentReleaseName
		spinner = ui.NewSpinner("Upgrading CRDs")
	} else {
		spinner.Success("CRDs not found")
		spinner = ui.NewSpinner("Installing CRDs")
	}

	opts := CreateCRDsHelmOptions(namespace, releaseName, dryRun, nil)
	common2.HandleCLIError(common2.HelmUpgradeOrInstallGeneric(opts))
	spinner.Success("CRDs installed")
}

func UiInstallAgent(cmd *cobra.Command, name string, defaultLabels []string, extraHelmValues ...map[string]interface{}) {
	autoCreate, _ := cmd.Flags().GetBool("create")
	ns, _ := cmd.Flags().GetString("namespace")
	executionNs, _ := cmd.Flags().GetString("execution-namespace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	floating, _ := cmd.Flags().GetBool("floating")
	globalTemplatePath, _ := cmd.Flags().GetString("global-template-path")
	isGlobalRunner, _ := cmd.Flags().GetBool("global")
	runnerGroup, _ := cmd.Flags().GetString("group")
	// Component flags
	executionChanged, enableExecution := common2.ExecutionCapabilityFromFlags(cmd)
	listenerChanged := cmd.Flags().Changed("listener")
	gitopsChanged := cmd.Flags().Changed("gitops")
	webhooksChanged := cmd.Flags().Changed("webhooks")
	anyChanged := executionChanged || listenerChanged || gitopsChanged || webhooksChanged
	enableListener, _ := cmd.Flags().GetBool("listener")
	enableGitops, _ := cmd.Flags().GetBool("gitops")
	enableWebhooks, _ := cmd.Flags().GetBool("webhooks")
	// we default to both capabilities if none flags are set
	if !anyChanged {
		enableExecution = true
		enableListener = true
	}

	var globalTemplate []byte
	if globalTemplatePath != "" {
		var err error
		globalTemplate, err = os.ReadFile(globalTemplatePath)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrInvalidRuntimeParameter,
				"Error reading the global template",
				"Check that the '--global-template-path' value points at a readable file",
				err,
			))
		}
		globalTemplateMap := make(map[string]interface{})
		err = yaml.Unmarshal(globalTemplate, &globalTemplateMap)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrInvalidRuntimeParameter,
				"Error parsing the global template",
				"The file that '--global-template-path' names must be a YAML Test Workflow template",
				err,
			))
		}
		if spec, ok := globalTemplateMap["spec"]; ok {
			globalTemplate, err = json.Marshal(spec)
		} else {
			globalTemplate, err = json.Marshal(globalTemplateMap)
		}
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrInvalidRuntimeParameter,
				"Error converting the global template",
				"The file that '--global-template-path' names must hold values that convert to JSON",
				err,
			))
		}
	}

	// Validate if the Agent exists
	var agent *cloudclient.Agent
	if name != "" {
		var err error
		agent, err = GetControlPlaneAgent(cmd, name)
		if err != nil && !autoCreate {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentGetFailed,
				"Error getting the agent",
				"Check the agent name or ID and that your credentials are valid, or pass '--create' to create the agent",
				err,
			))
		}
		if agent != nil {
			PrintControlPlaneAgent(*agent)
			ui.NL()
		}
	}

	// Create new Agent if it's expected
	if agent == nil && autoCreate {
		labels, _ := cmd.Flags().GetStringSlice("label")
		labels = append(labels, defaultLabels...)
		environmentIds, _ := cmd.Flags().GetStringSlice("env")
		agent = UiCreateAgent(
			cmd,
			name,
			labels,
			environmentIds,
			isGlobalRunner,
			runnerGroup,
			floating,
			enableExecution,
			enableListener,
			enableGitops,
			enableWebhooks,
		)
	}

	// Load agents from the Control Plane and select one
	if agent == nil {
		agents, err := GetControlPlaneAgents(cmd, false)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentGetFailed,
				"Error getting the agents",
				common2.AgentLookupHint,
				err,
			))
		}

		if name == "" {
			name = ui.Select("select agent", common.MapSlice(agents, func(t cloudclient.Agent) string {
				return t.Name
			}))
			if name == "" {
				common2.HandleCLIError(common2.NewCLIError(
					common2.TKErrInvalidRuntimeParameter,
					"No agent name provided",
					"Pass the agent name as an argument, for example `testkube install agent my-agent`",
					fmt.Errorf("agent name not provided"),
				))
			}
		}

		for _, a := range agents {
			if a.Name == name || a.ID == name {
				agent = &a
				break
			}
		}
	}

	// Fail if there is no matching agent available
	if agent == nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrResourceNotFound,
			"Agent not found",
			"Check the agent name or ID and list the agents with `testkube get agents`, or pass '--create' to create it",
			fmt.Errorf("agent %s not found", name),
		))
		return
	}

	if secretKey, _ := cmd.Flags().GetString("secret"); agent.SecretKey == "" && secretKey != "" {
		agent.SecretKey = secretKey
	}

	if agent.SecretKey == "" {
		secretKey, err := GetControlPlaneAgentSecretKey(cmd, agent.ID)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentGetFailed,
				"Error getting the agent secret key",
				"Check that your credentials are valid and that your user can read the secret key of this agent, or pass it with '--secret'",
				err,
			))
		}
		agent.SecretKey = secretKey
	}

	// Auto-detect the namespace
	if ns == "" {
		var nses []string
		if agent.Namespace == "" {
			nses, _ = GetKubernetesNamespaces()
		} else {
			nses = []string{agent.Namespace}
		}
		existingAgents, err := GetKubernetesAgents(nses)
		if err == nil {
			for _, ag := range existingAgents {
				if ag.Pod.Namespace != "" && ag.AgentID.Value == agent.ID {
					ns = ag.Pod.Namespace
					ui.Warn("Detected existing installation in namespace", ns)
					break
				}
			}
		}
	}

	if ns == "" {
		defaultNs := agent.Namespace
		if defaultNs == "" {
			defaultNs = agent.Name
		}
		ns = ui.TextInput("namespace to install", defaultNs)
		if ns == "" {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrInvalidRuntimeParameter,
				"No namespace provided",
				"Pass the namespace with '--namespace', or type one at the prompt",
				fmt.Errorf("you need to select namespace to install"),
			))
		}
	}

	// Load the Cloud settings
	cfg, err := config.Load()
	if err != nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrConfigInitFailed,
			"Error loading testkube config file",
			common2.ConfigFileHint,
			err,
		))
	}
	skipTLS := common2.ResolveSkipTLS(cmd, &cfg)
	opts := &common2.HelmOptions{}
	common2.ProcessMasterFlags(cmd, opts, &cfg)

	agentUri := opts.Master.URIs.Agent
	if cfg.CloudContext.AgentUri != "" {
		agentUri = regexp.MustCompile("^[^:]+://").ReplaceAllString(cfg.CloudContext.AgentUri, "")
	}
	agentSecure := strings.HasPrefix(cfg.CloudContext.AgentUri, "https://") || regexp.MustCompile(":(6)?443$").MatchString(agentUri) || !strings.Contains(agentUri, ":")

	controlPlane := ControlPlaneConfig{
		URL:            agentUri,
		Secure:         agentSecure,
		SkipVerify:     skipTLS,
		OrganizationID: cfg.CloudContext.OrganizationId,
		EnvironmentID:  cfg.CloudContext.EnvironmentId,
		Agent:          *agent,
	}
	version, _ := cmd.Flags().GetString("version")

	var spinner *pterm.SpinnerPrinter
	if !dryRun {
		spinner = ui.NewSpinner("Running Helm command...")
	}

	// Install runner chart
	helmOpts := CreateRunnerHelmOptions(controlPlane, ns, version, dryRun, map[string]interface{}{
		"runner.enabled":   enableExecution,
		"listener.enabled": enableListener,
		"gitops.enabled":   enableGitops,
		"webhooks.enabled": enableWebhooks,
	})
	// Apply caller-provided extra Helm values (e.g. pro connect disabling CRD install)
	for _, extra := range extraHelmValues {
		for k, v := range extra {
			helmOpts.Values[k] = v
		}
	}
	// When listener capability is enabled, pass the environment ID to the runner
	if enableListener && controlPlane.EnvironmentID != "" {
		helmOpts.Values["runner.envId"] = controlPlane.EnvironmentID
	}
	if executionNs != "" && executionNs != ns {
		helmOpts.Values["execution.default.namespace"] = executionNs
	}
	if len(globalTemplate) > 0 {
		helmOpts.Values["globalTemplate.enabled"] = true
		helmOpts.Values["globalTemplate.inline"] = true
		helmOpts.Values["globalTemplate.spec"] = string(globalTemplate)
	}
	common2.HandleCLIError(common2.HelmUpgradeOrInstallGeneric(helmOpts))

	if dryRun {
		return
	}

	spinner.Success()

	agents, err := GetKubernetesAgents([]string{ns})
	if err != nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrResourceLookupFailed,
			"Error getting the agents running in the cluster",
			common2.ClusterLookupHint,
			err,
		))
	}

	var foundAgent *internalAgent
	for i := range agents {
		if agents[i].AgentID.Value == agent.ID {
			foundAgent = &agents[i]
			break
		}
	}

	if foundAgent == nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrResourceNotFound,
			"Agent not found in the cluster",
			"The Helm release installed, but no agent Pod carries its id yet. Check the Pods of the namespace, for example with `kubectl get pods -n <namespace>`",
			fmt.Errorf("not found the agent installed in namespace '%s'", ns),
		))
		return
	}

	PrintKubernetesAgent(*foundAgent)
}
