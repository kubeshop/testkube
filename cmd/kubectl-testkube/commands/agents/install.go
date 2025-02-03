package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/common"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallAgentCommand() *cobra.Command {
	var (
		secretKey string
		namespace string
	)

	cmd := &cobra.Command{
		Use:    "agent <name>",
		Args:   cobra.MaximumNArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			UiInstallAgent(cmd, strings.Join(args, ""), "")
		},
	}

	// Installation
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")

	// Install existing
	cmd.Flags().StringVarP(&secretKey, "secret", "s", "", "secret key for the selected agent")

	return cmd
}

func NewInstallRunnerCommand() *cobra.Command {
	var (
		secretKey string

		namespace string
		version   string
		dryRun    bool

		globalTemplatePath string

		autoCreate     bool
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:    "runner <name>",
		Args:   cobra.MaximumNArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			UiInstallAgent(cmd, strings.Join(args, ""), "runner")
		},
	}

	// Installation > General
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")
	cmd.Flags().StringVar(&version, "version", "", "agent version to use (defaults to latest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "display helm commands only")

	// Installation > Runner
	cmd.Flags().StringVarP(&globalTemplatePath, "global-template-path", "g", "", "include global template")

	// Install existing
	cmd.Flags().StringVarP(&secretKey, "secret", "s", "", "secret key for the selected agent")

	// Create and install
	cmd.Flags().BoolVar(&autoCreate, "create", false, "auto create that agent")
	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "(with --create) environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "(with --create) label key value pair: --label key1=value1")

	return cmd
}

func NewInstallGitOpsCommand() *cobra.Command {
	var (
		secretKey string
		dryRun    bool

		namespace             string
		fromCloud             bool
		toCloud               bool
		cloudNamePattern      string
		kubernetesNamePattern string
		version               string

		autoCreate     bool
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:    "gitops <name>",
		Args:   cobra.MaximumNArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			UiInstallAgent(cmd, strings.Join(args, ""), "gitops")
		},
	}

	// Installation > General
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to install the agent")
	cmd.Flags().StringVar(&version, "version", "", "agent version to use (defaults to latest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "display helm commands only")

	// Installation > GitOps
	cmd.Flags().BoolVarP(&fromCloud, "from-cloud", "f", false, "should synchronize data from cloud")
	cmd.Flags().BoolVarP(&toCloud, "to-cloud", "t", false, "should synchronize data to cloud")
	cmd.Flags().StringVarP(&cloudNamePattern, "cloud-pattern", "F", "<name>", "pattern for resource names in cloud")
	cmd.Flags().StringVarP(&kubernetesNamePattern, "kubernetes-pattern", "T", "<name>", "pattern for resource names in kubernetes")

	// Install existing
	cmd.Flags().StringVarP(&secretKey, "secret", "s", "", "secret key for the selected agent")

	// Create and install
	cmd.Flags().BoolVar(&autoCreate, "create", false, "auto create that agent")
	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "(with --create) environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "(with --create) label key value pair: --label key1=value1")

	return cmd
}

func UiInstallAgent(cmd *cobra.Command, name string, agentType string) {
	autoCreate, _ := cmd.Flags().GetBool("create")
	ns, _ := cmd.Flags().GetString("namespace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	agentType, _ = GetInternalAgentType(agentType)
	globalTemplatePath, _ := cmd.Flags().GetString("global-template-path")
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
		agent = UiCreateAgent(cmd, agentType, name, labels, environmentIds)
	}

	// Load agents from the Control Plane and select one
	if agent == nil {
		agents, err := GetControlPlaneAgents(cmd, agentType)
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

	spinner := ui.NewSpinner("Running Helm command...")

	switch agentType {
	case "sync":
		if len(agent.Environments) == 0 {
			spinner.Fail("could not select environment for this GitOps Agent")
			os.Exit(1)
		}

		fromCloud, _ := cmd.Flags().GetBool("from-cloud")
		toCloud, _ := cmd.Flags().GetBool("to-cloud")
		cloudPattern, _ := cmd.Flags().GetString("cloud-pattern")
		kubernetesPattern, _ := cmd.Flags().GetString("kubernetes-pattern")
		opts := CreateHelmOptions(controlPlane, ns, version, dryRun, map[string]interface{}{
			"testkube-api.next.gitops.syncCloudToKubernetes":   fromCloud,
			"testkube-api.next.gitops.syncKubernetesToCloud":   toCloud,
			"testkube-api.next.gitops.namePatterns.cloud":      cloudPattern,
			"testkube-api.next.gitops.namePatterns.kubernetes": kubernetesPattern,
		})
		cliErr := common2.HelmUpgradeOrInstallGeneric(opts)
		if cliErr != nil {
			cliErr.Print()
			os.Exit(1)
		}
	case "run":
		opts := CreateHelmOptions(controlPlane, ns, version, dryRun, map[string]interface{}{
			"testkube-api.next.runner.enabled": true,
		})
		if len(globalTemplate) > 0 {
			opts.Values["global.testWorkflows.globalTemplate.enabled"] = true
			opts.Values["global.testWorkflows.globalTemplate.inline"] = true
			opts.Values["global.testWorkflows.globalTemplate.spec"] = string(globalTemplate)
		}
		cliErr := common2.HelmUpgradeOrInstallGeneric(opts)
		if cliErr != nil {
			cliErr.Print()
			os.Exit(1)
		}
	default:
		spinner.Fail(fmt.Sprintf("unknown agent type %s", agentType))
		os.Exit(1)
	}
	spinner.Success()

	agents, err := GetKubernetesAgents([]string{ns})
	ui.ExitOnError("getting agents in kubernetes")

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
