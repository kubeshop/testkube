package agents

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/common"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/ui"
)

type internalAgent struct {
	Pod       corev1.Pod
	Container corev1.Container
	Ready     bool
	Detected  bool

	AgentID        corev1.EnvVar
	EnvironmentID  corev1.EnvVar
	OrganizationID corev1.EnvVar
	CloudURL       corev1.EnvVar

	Registered *cloudclient.Agent
}

var agentLabelMap = map[string]string{
	"run":  "Runner",
	"sync": "GitOps",
	"agnt": "SuperAgent",
}

var agentKnownTypeMap = map[string]string{
	"gitops":     "sync",
	"runner":     "run",
	"superagent": "agnt",
}

func GetInternalAgentType(name string) (string, error) {
	name = strings.ToLower(name)
	for k, v := range agentKnownTypeMap {
		if v == name || k == name {
			return v, nil
		}
	}
	return name, errors.New("unknown")
}

type internalAgents []internalAgent

func (list internalAgents) Table() (header []string, output [][]string) {
	header = []string{"Type", "Name", "Version", "Namespace", "Environments", "Labels"}
	for _, e := range list {
		agentType := "-"
		agentVersion := "-"
		agentName := e.AgentID.Value
		agentEnvironments := "-"
		agentLabels := "-"
		namespace := e.Pod.Namespace
		if e.Registered != nil {
			agentType = e.Registered.Type
			agentName = e.Registered.Name
			agentVersion = e.Registered.Version
			agentEnvironments = strings.Join(common.MapSlice(e.Registered.Environments, func(t cloudclient.AgentEnvironment) string {
				if t.Name == "" {
					return t.ID
				}
				return t.Name
			}), ", ")
			agentLabelsEntries := make([]string, 0)
			for k, v := range e.Registered.Labels {
				agentLabelsEntries = append(agentLabelsEntries, fmt.Sprintf("%s=%s", k, v))
			}
			agentLabels = strings.Join(agentLabelsEntries, " ")
		}
		if agentLabelMap[agentType] != "" {
			agentType = agentLabelMap[agentType]
		}
		if e.Detected {
			if e.Ready {
				namespace = fmt.Sprintf("%s:%s", ui.Green("•"), namespace)
			} else {
				namespace = fmt.Sprintf("%s:%s", ui.Red("•"), namespace)
			}
		} else if e.Registered != nil {
			namespace = e.Registered.Namespace
		}
		output = append(output, []string{
			agentType,
			agentName,
			agentVersion,
			namespace,
			agentEnvironments,
			agentLabels,
		})
	}
	return
}

func GetKubernetesNamespaces() ([]string, error) {
	kubeClient, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, errors.Wrap(err, "connecting to kubernetes")
	}

	listNs, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "listing namespaces in kubernetes")
	}
	return common.MapSlice(listNs.Items, func(ns corev1.Namespace) string {
		return ns.Name
	}), nil
}

func GetControlPlaneEnvironments(cmd *cobra.Command) (map[string]cloudclient.Environment, error) {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return nil, errors.New("no api key found in config")
	}

	envs, err := common2.GetEnvironments(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId)
	if err != nil {
		return nil, errors.Wrap(err, "getting environments")
	}
	envsMap := make(map[string]cloudclient.Environment)
	for _, env := range envs {
		envsMap[env.Id] = env
	}
	return envsMap, nil
}

func EnableNewArchitecture(cmd *cobra.Command, env cloudclient.Environment) error {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return errors.New("no api key found in config")
	}

	return common2.EnableNewArchitecture(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, env)
}

func GetControlPlaneAgents(cmd *cobra.Command, agentType string) ([]cloudclient.Agent, error) {
	agentType, _ = GetInternalAgentType(agentType)
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return nil, errors.New("no api key found in config")
	}

	registeredAgents, err := common2.GetAgents(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, agentType)
	if err != nil {
		return nil, errors.Wrap(err, "getting agents")
	}

	return registeredAgents, nil
}

func GetControlPlaneAgent(cmd *cobra.Command, idOrName string) (*cloudclient.Agent, error) {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return nil, errors.New("no api key found in config")
	}

	agent, err := common2.GetAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, idOrName)
	if err != nil {
		return nil, errors.Wrap(err, "getting agent")
	}
	return &agent, nil
}

func DeleteControlPlaneAgent(cmd *cobra.Command, idOrName string) error {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return errors.New("no api key found in config")
	}

	err = common2.DeleteAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, idOrName)
	if err != nil {
		return errors.Wrap(err, "getting agent")
	}
	return nil
}

func GetControlPlaneAgentSecretKey(cmd *cobra.Command, idOrName string) (string, error) {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return "", errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return "", errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return "", errors.New("no api key found in config")
	}

	secretKey, err := common2.GetAgentSecretKey(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, idOrName)
	if err != nil {
		return "", errors.Wrap(err, "getting secret key")
	}
	return secretKey, nil
}

func CreateAgent(cmd *cobra.Command, input cloudclient.AgentInput) (*cloudclient.Agent, error) {
	_, _, err := common2.GetClient(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to cloud")
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}
	if cfg.CloudContext.ApiKey == "" {
		return nil, errors.New("no api key found in config")
	}

	agent, err := common2.CreateAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, input)
	if err != nil {
		return nil, errors.Wrap(err, "creating agent")
	}
	return &agent, nil
}

func GetKubernetesAgents(namespaces []string) (internalAgents, error) {
	kubeClient, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, errors.Wrap(err, "connecting to kubernetes")
	}

	// Get the available agents
	var agents []internalAgent
	for _, ns := range namespaces {
		pods, err := kubeClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "listing pods in kubernetes")
		}

		// Detect Agents
		// TODO: Handle current status + IDs from secrets [via health API]
	loop:
		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				for _, env := range container.Env {
					if env.Name == "TESTKUBE_PRO_AGENT_ID" || env.Name == "TESTKUBE_CLOUD_AGENT_ID" {
						a := internalAgent{Pod: pod, Container: container, Detected: true}
						for _, c := range a.Pod.Status.Conditions {
							if c.Type == corev1.PodReady {
								a.Ready = c.Status == corev1.ConditionTrue
							}
						}
						for _, env := range a.Container.Env {
							switch env.Name {
							case "TESTKUBE_PRO_AGENT_ID", "TESTKUBE_CLOUD_AGENT_ID":
								a.AgentID = env
							case "TESTKUBE_PRO_ENV_ID", "TESTKUBE_CLOUD_ENV_ID":
								a.EnvironmentID = env
							case "TESTKUBE_PRO_ORG_ID", "TESTKUBE_CLOUD_ORG_ID":
								a.OrganizationID = env
							case "TESTKUBE_PRO_URL", "TESTKUBE_CLOUD_URL":
								a.CloudURL = env
							}
						}

						agents = append(agents, a)
						continue loop
					}
				}
			}
		}
	}

	return agents, nil
}

func CombineAgents(kubernetesAgents internalAgents, controlPlaneAgents []cloudclient.Agent) internalAgents {
	controlPlaneAgentsMap := make(map[string]*cloudclient.Agent)
	found := make(map[string]bool)
	for i := range controlPlaneAgents {
		controlPlaneAgentsMap[controlPlaneAgents[i].ID] = &controlPlaneAgents[i]
	}

	for i := range kubernetesAgents {
		if kubernetesAgents[i].AgentID.Value != "" {
			kubernetesAgents[i].Registered = controlPlaneAgentsMap[kubernetesAgents[i].AgentID.Value]
			found[kubernetesAgents[i].AgentID.Value] = true
		}
	}

	for i := range controlPlaneAgents {
		if !found[controlPlaneAgents[i].ID] {
			kubernetesAgents = append(kubernetesAgents, internalAgent{
				Registered: &controlPlaneAgents[i],
			})
		}
	}

	return kubernetesAgents
}

type ControlPlaneConfig struct {
	URL            string
	Secure         bool
	OrganizationID string
	EnvironmentID  string
	Agent          cloudclient.Agent
}

func CreateHelmOptions(
	controlPlane ControlPlaneConfig,
	installationNamespace string,
	version string,
	dryRun bool,
	additionalValues map[string]interface{},
) common2.HelmGenericOptions {
	envId := ""
	for _, v := range controlPlane.Agent.Environments {
		if v.ID == controlPlane.EnvironmentID {
			envId = v.ID
		}
	}
	if envId == "" && len(controlPlane.Agent.Environments) == 1 {
		envId = controlPlane.Agent.Environments[0].ID
	}

	values := map[string]interface{}{
		// Creating initial templates
		"global.testWorkflows.createServiceAccountTemplates": false,
		"global.testWorkflows.createOfficialTemplates":       false,
		"global.testWorkflows.globalTemplate.enabled":        false,

		// Setting the names
		"testkube-api.fullnameOverride": fmt.Sprintf("testkube-%s", controlPlane.Agent.Name),

		// Setting the connection
		"testkube-api.cloud.url":         controlPlane.URL,
		"testkube-api.cloud.tls.enabled": controlPlane.Secure,
		"testkube-api.cloud.key":         controlPlane.Agent.SecretKey,
		"testkube-api.cloud.agentId":     controlPlane.Agent.ID,
		"testkube-api.cloud.orgId":       controlPlane.OrganizationID,
		"testkube-api.cloud.envId":       envId,

		// Disabling unnecessary features
		"testkube-api.nats.enabled":    false,
		"testkube-api.nats.embedded":   true,
		"testkube-api.minio.enabled":   false,
		"testkube-api.enableK8sEvents": false,
		"mongodb.enabled":              false,
		"testkube-operator.enabled":    false, // TODO: INSTALL CRDs

		// Enable GitOps runner
		"testkube-api.next.enabled":                        true,
		"testkube-api.next.cloudStorage":                   true,
		"testkube-api.next.gitops.syncCloudToKubernetes":   false,
		"testkube-api.next.gitops.syncKubernetesToCloud":   false,
		"testkube-api.next.gitops.namePatterns.cloud":      false,
		"testkube-api.next.gitops.namePatterns.kubernetes": false,
		"testkube-api.next.legacyAgent.enabled":            false,
		"testkube-api.next.webhooks.enabled":               false,
		"testkube-api.next.testTriggers.enabled":           false,
		"testkube-api.next.runner.enabled":                 false,
		"testkube-api.next.legacyTests.enabled":            false,
	}
	maps.Copy(values, additionalValues)
	if version != "" {
		values["testkube-api.image.tag"] = version
		values["testkube-api.twToolkitImage.tag"] = version
		values["testkube-api.twInitImage.tag"] = version
	}
	return common2.HelmGenericOptions{
		DryRun:         dryRun,
		RegistryURL:    "https://kubeshop.github.io/helm-charts",
		RepositoryName: "kubeshop",
		ChartName:      "testkube",
		ReleaseName:    fmt.Sprintf("testkube-%s", controlPlane.Agent.Name),

		Namespace: installationNamespace,
		Args:      []string{"--wait"},
		Values:    values,
	}
}

func PrintControlPlaneAgent(agent cloudclient.Agent) {
	agentTypeLabel := agent.Type
	if agentLabelMap[agentTypeLabel] != "" {
		agentTypeLabel = agentLabelMap[agentTypeLabel]
	}
	agentSecretKey := agent.SecretKey
	if agent.SecretKey == "" {
		agentSecretKey = ui.LightGray("<encrypted>")
	}
	ui.Warn("ID:            ", agent.ID)
	ui.Warn("Secret Key:    ", agentSecretKey)
	ui.Warn("Type:          ", agentTypeLabel)
	ui.Warn("Name:          ", agent.Name)
	ui.Warn("Created:       ", agent.CreatedAt.In(time.Local).Format(time.RFC822Z)+" "+ui.LightGray("("+time.Since(agent.CreatedAt).Truncate(time.Second).String()+")"))
	if agent.AccessedAt == nil {
		ui.Warn("Last Activity: ", ui.LightGray("never"))
	} else {
		ui.Warn("Last Activity: ", agent.AccessedAt.In(time.Local).Format(time.RFC822Z)+" "+ui.LightGray("("+time.Since(*agent.AccessedAt).Truncate(time.Second).String()+")"))
	}
	ui.Warn("Last Version:  ", agent.Version)
	ui.Warn("Last Namespace:", agent.Namespace)
	ui.Warn("Environments:")
	for _, env := range agent.Environments {
		fmt.Println("   ", env.Name, ui.LightGray("("+env.ID+")"))
	}
	if len(agent.Environments) == 0 {
		fmt.Println("   none")
	}
	ui.Warn("Labels:")
	maxLabelSize := 0
	for k := range agent.Labels {
		if len(k) > maxLabelSize {
			maxLabelSize = len(k)
		}
	}
	for k, v := range agent.Labels {
		ui.Warn("   "+strings.Repeat(" ", maxLabelSize-len(k))+k, ui.LightGray("= ")+v)
	}
	if len(agent.Labels) == 0 {
		fmt.Println("   none")
	}
}

func PrintKubernetesAgent(agent internalAgent) {
	ui.Warn("Pod Name:", agent.Pod.Name)
	ui.Warn("Ready:   ", fmt.Sprintf("%v", agent.Ready))
}
