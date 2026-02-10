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
	var (
		secretKey string

		namespace          string
		executionNamespace string
		version            string
		dryRun             bool

		globalTemplatePath string
		global             bool
		group              string

		autoCreate     bool
		floating       bool
		labelPairs     []string
		environmentIds []string

		runner    bool
		listener  bool
		gitops    bool
		webhooks  bool
		agentType string
	)
	cmd := &cobra.Command{
		Use:  "agent <name>",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check for deprecated --type flag usage
			if cmd.Flags().Changed("type") {
				ui.Warn("⚠️  The --type/-t flag is deprecated.")
				ui.Info("Please use capability flags instead:")
				ui.Info("  --runner    : Enable runner capability")
				ui.Info("  --listener  : Enable listener capability")
				ui.Info("  --gitops    : Enable GitOps capability")
				ui.Info("  --webhooks  : Enable webhooks capability")
				ui.NL()
				return
			}

			UiInstallAgent(cmd, strings.Join(args, ""))
		},
	}

	// Installation > General
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")
	cmd.Flags().StringVarP(&executionNamespace, "execution-namespace", "N", "", "namespace to run executions (defaults to installation namespace)")
	cmd.Flags().StringVar(&version, "version", "", "agent version to use (defaults to latest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "display helm commands only")

	// Installation > Runner
	cmd.Flags().StringVarP(&globalTemplatePath, "global-template-path", "g", "", "include global template")
	cmd.Flags().BoolVar(&global, "global", false, "make it global agent")
	cmd.Flags().StringVar(&group, "group", "", "make it grouped agent")

	// Install existing
	cmd.Flags().StringVarP(&secretKey, "secret", "s", "", "secret key for the selected agent")

	// Create and install
	cmd.Flags().BoolVar(&autoCreate, "create", false, "auto create that agent")
	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "(with --create) environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "(with --create) label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&floating, "floating", false, "(with --create) create as a floating agent")

	// Components selection
	cmd.Flags().BoolVar(&runner, "runner", false, "enable runner component (default: enabled when no component flags are set)")
	cmd.Flags().BoolVar(&listener, "listener", false, "enable listener component (default: enabled when no component flags are set)")
	cmd.Flags().BoolVar(&gitops, "gitops", false, "enable gitops capability")
	cmd.Flags().BoolVar(&webhooks, "webhooks", false, "enable webhooks capability")

	// Deprecated flag
	cmd.Flags().StringVarP(&agentType, "type", "t", "", "[DEPRECATED] agent type - use capability flags instead")
	cmd.Flags().MarkDeprecated("type", "use --runner, --listener, --gitops, and/or --webhooks instead")

	return cmd
}

// NewInstallRunnerCommand creates a command equivalent to `install agent --runner`.
// It intentionally does not expose the --listener flag.
func NewInstallRunnerCommand() *cobra.Command {
	var (
		secretKey string

		namespace          string
		executionNamespace string
		version            string
		dryRun             bool

		globalTemplatePath string
		global             bool
		group              string

		autoCreate     bool
		floating       bool
		labelPairs     []string
		environmentIds []string

		runner    bool
		agentType string
	)

	cmd := &cobra.Command{
		Use:  "runner <name>",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check for deprecated --type flag usage
			if cmd.Flags().Changed("type") {
				ui.Warn("⚠️  The --type/-t flag is deprecated.")
				ui.Info("This command installs a runner-only agent by default.")
				ui.Info("For more flexibility, use 'kubectl testkube install agent' with:")
				ui.Info("  --runner    : Enable runner capability")
				ui.Info("  --listener  : Enable listener capability")
				ui.Info("  --gitops    : Enable GitOps capability")
				ui.Info("  --webhooks  : Enable webhooks capability")
				ui.NL()
				return
			}

			// Force runner-only behavior
			_ = cmd.Flags().Set("runner", "true")
			UiInstallAgent(cmd, strings.Join(args, ""))
		},
	}

	// Installation > General
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")
	cmd.Flags().StringVarP(&executionNamespace, "execution-namespace", "N", "", "namespace to run executions (defaults to installation namespace)")
	cmd.Flags().StringVar(&version, "version", "", "agent version to use (defaults to latest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "display helm commands only")

	// Installation > Runner
	cmd.Flags().StringVarP(&globalTemplatePath, "global-template-path", "g", "", "include global template")
	cmd.Flags().BoolVar(&global, "global", false, "make it global agent")
	cmd.Flags().StringVar(&group, "group", "", "make it grouped agent")

	// Install existing
	cmd.Flags().StringVarP(&secretKey, "secret", "s", "", "secret key for the selected agent")

	// Create and install
	cmd.Flags().BoolVar(&autoCreate, "create", false, "auto create that agent")
	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "(with --create) environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "(with --create) label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&floating, "floating", false, "(with --create) create as a floating agent")

	// Component selection: only runner; keep flag hidden and force it on
	cmd.Flags().BoolVar(&runner, "runner", true, "enable runner component")
	_ = cmd.Flags().MarkHidden("runner")

	// Deprecated flag
	cmd.Flags().StringVarP(&agentType, "type", "t", "", "[DEPRECATED] agent type - this command installs runner-only agents")
	cmd.Flags().MarkDeprecated("type", "this command installs runner-only agents by default")

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
		spinner.Fail(err)
		os.Exit(1)
	}

	if installed && currentReleaseName == "" {
		spinner.Fail("The CRDs are installed, but they are not managed by our Helm Chart")
		os.Exit(1)
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
	cliErr := common2.HelmUpgradeOrInstallGeneric(opts)
	if cliErr != nil {
		cliErr.Print()
		os.Exit(1)
	}
	spinner.Success("CRDs installed")
}

func UiInstallAgent(cmd *cobra.Command, name string) {
	autoCreate, _ := cmd.Flags().GetBool("create")
	ns, _ := cmd.Flags().GetString("namespace")
	executionNs, _ := cmd.Flags().GetString("execution-namespace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	floating, _ := cmd.Flags().GetBool("floating")
	globalTemplatePath, _ := cmd.Flags().GetString("global-template-path")
	isGlobalRunner, _ := cmd.Flags().GetBool("global")
	runnerGroup, _ := cmd.Flags().GetString("group")
	// Component flags
	runnerChanged := cmd.Flags().Changed("runner")
	listenerChanged := cmd.Flags().Changed("listener")
	gitopsChanged := cmd.Flags().Changed("gitops")
	webhooksChanged := cmd.Flags().Changed("webhooks")
	anyChanged := runnerChanged || listenerChanged || gitopsChanged || webhooksChanged
	enableRunner, _ := cmd.Flags().GetBool("runner")
	enableListener, _ := cmd.Flags().GetBool("listener")
	enableGitops, _ := cmd.Flags().GetBool("gitops")
	enableWebhooks, _ := cmd.Flags().GetBool("webhooks")
	// we default to both capabilities if none flags are set
	if !anyChanged {
		enableRunner = true
		enableListener = true
	}

	var globalTemplate []byte
	if globalTemplatePath != "" {
		var err error
		globalTemplate, err = os.ReadFile(globalTemplatePath)
		ui.ExitOnError("reading global template", err)
		globalTemplateMap := make(map[string]interface{})
		err = yaml.Unmarshal(globalTemplate, &globalTemplateMap)
		ui.ExitOnError("reading global template", err)
		if spec, ok := globalTemplateMap["spec"]; ok {
			globalTemplate, err = json.Marshal(spec)
			ui.ExitOnError("marshalling global template", err)
		} else {
			globalTemplate, err = json.Marshal(globalTemplateMap)
			ui.ExitOnError("marshalling global template", err)
		}
	}

	// Validate if the Agent exists
	var agent *cloudclient.Agent
	if name != "" {
		var err error
		agent, err = GetControlPlaneAgent(cmd, name)
		if !autoCreate {
			ui.ExitOnError("getting agent", err)
		}
		if agent != nil {
			PrintControlPlaneAgent(*agent)
			ui.NL()
		}
	}

	// Create new Agent if it's expected
	if agent == nil && autoCreate {
		labels, _ := cmd.Flags().GetStringSlice("label")
		environmentIds, _ := cmd.Flags().GetStringSlice("env")
		agent = UiCreateAgent(
			cmd,
			name,
			labels,
			environmentIds,
			isGlobalRunner,
			runnerGroup,
			floating,
			enableRunner,
			enableListener,
			enableGitops,
			enableWebhooks,
		)
	}

	// Load agents from the Control Plane and select one
	if agent == nil {
		agents, err := GetControlPlaneAgents(cmd, false)
		ui.ExitOnError("listing agents", err)

		if name == "" {
			name = ui.Select("select agent", common.MapSlice(agents, func(t cloudclient.Agent) string {
				return t.Name
			}))
			if name == "" {
				ui.Failf("agent name not provided")
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
		ui.Failf("agent %s not found", name)
	}

	if secretKey, _ := cmd.Flags().GetString("secret"); agent.SecretKey == "" && secretKey != "" {
		agent.SecretKey = secretKey
	}

	if agent.SecretKey == "" {
		secretKey, err := GetControlPlaneAgentSecretKey(cmd, agent.ID)
		ui.ExitOnError("failed to fetch the secret key", err)
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
			ui.Failf("you need to select namespace to install")
		}
	}

	// Load the Cloud settings
	cfg, err := config.Load()
	ui.ExitOnError("loading config file", err)
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
		"runner.enabled":   enableRunner,
		"listener.enabled": enableListener,
		"gitops.enabled":   enableGitops,
		"webhooks.enabled": enableWebhooks,
	})
	// When listener capability is enabled, pass the environment ID to the runner
	if enableListener && controlPlane.EnvironmentID != "" {
		helmOpts.Values["runner.envId"] = controlPlane.EnvironmentID
	}
	if executionNs != "" && executionNs != ns {
		helmOpts.Values["execution.default.namespace"] = executionNs
	}
	if len(globalTemplate) > 0 {
		helmOpts.Values["globalTemplate.enabled"] = true
		helmOpts.Values["globalTemplate.spec"] = string(globalTemplate)
	}
	cliErr := common2.HelmUpgradeOrInstallGeneric(helmOpts)
	if cliErr != nil {
		cliErr.Print()
		os.Exit(1)
	}

	if dryRun {
		return
	}

	spinner.Success()

	agents, err := GetKubernetesAgents([]string{ns})
	ui.ExitOnError("getting agents in kubernetes", err)

	var foundAgent *internalAgent
	for i := range agents {
		if agents[i].AgentID.Value == agent.ID {
			foundAgent = &agents[i]
			break
		}
	}

	if foundAgent == nil {
		ui.Failf("not found the agent installed in namespace '%s'", ns)
	}

	PrintKubernetesAgent(*foundAgent)
}
