package agents

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
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

type internalAgents []internalAgent

func (list internalAgents) Table() (header []string, output [][]string) {
	header = []string{"Name", "Version", "Namespace", "Environments", "Labels"}
	for _, e := range list {
		agentVersion := "-"
		agentName := e.AgentID.Value
		agentEnvironments := "-"
		agentLabels := "-"
		namespace := e.Pod.Namespace
		if e.Registered != nil {
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

func GetControlPlaneAgents(cmd *cobra.Command, includeDeleted bool) ([]cloudclient.Agent, error) {
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

	registeredAgents, err := common2.GetAgents(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, "run", includeDeleted)
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

func UpdateAgent(cmd *cobra.Command, idOrName string, input cloudclient.AgentInput) (*cloudclient.Agent, error) {
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

	err = common2.UpdateAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, idOrName, input)
	if err != nil {
		return nil, errors.Wrap(err, "updating agent")
	}
	agent, err := common2.GetAgent(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, idOrName)
	if err != nil {
		return nil, errors.Wrap(err, "getting updated agent")
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
		"fullnameOverride": fmt.Sprintf("testkube-%s", controlPlane.Agent.Name),

		// Setting the connection
		"cloud.url":         controlPlane.URL,
		"cloud.tls.enabled": controlPlane.Secure,
		"cloud.key":         controlPlane.Agent.SecretKey,
		"cloud.agentId":     controlPlane.Agent.ID,
		"cloud.orgId":       controlPlane.OrganizationID,
		"cloud.envId":       envId,

		// Disabling unnecessary features
		"nats.enabled":    false,
		"nats.embedded":   true,
		"minio.enabled":   false,
		"enableK8sEvents": false,

		// Enable runner
		"multinamespace.enabled":    true, // TODO: Make its behavior default on next.enabled?
		"next.enabled":              true,
		"next.cloudStorage":         true,
		"next.legacyAgent.enabled":  false,
		"next.webhooks.enabled":     false,
		"next.testTriggers.enabled": false,
		"next.runner.enabled":       false,
		"next.legacyTests.enabled":  false,
	}
	maps.Copy(values, additionalValues)
	if version != "" {
		values["image.tag"] = version
		values["twToolkitImage.tag"] = version
		values["twInitImage.tag"] = version
	}
	return common2.HelmGenericOptions{
		DryRun:         dryRun,
		RegistryURL:    "https://kubeshop.github.io/helm-charts",
		RepositoryName: "kubeshop",
		ChartName:      "testkube-api",
		ReleaseName:    fmt.Sprintf("testkube-%s", controlPlane.Agent.Name),

		Namespace: installationNamespace,
		Args:      []string{"--wait"},
		Values:    values,
	}
}

func CreateRunnerHelmOptions(
	controlPlane ControlPlaneConfig,
	installationNamespace string,
	version string,
	dryRun bool,
	additionalValues map[string]interface{},
) common2.HelmGenericOptions {
	values := map[string]interface{}{
		// Setting the connection
		"runner.secret":     controlPlane.Agent.SecretKey,
		"runner.orgId":      controlPlane.OrganizationID,
		"runner.id":         controlPlane.Agent.ID,
		"cloud.url":         controlPlane.URL,
		"cloud.tls.enabled": controlPlane.Secure,
	}
	maps.Copy(values, additionalValues)
	if version != "" {
		values["images.agent.tag"] = version
		values["images.toolkit.tag"] = version
		values["images.init.tag"] = version
	}
	return common2.HelmGenericOptions{
		DryRun:         dryRun,
		RegistryURL:    "https://kubeshop.github.io/helm-charts",
		RepositoryName: "kubeshop",
		ChartName:      "testkube-runner",
		ReleaseName:    fmt.Sprintf("testkube-%s", controlPlane.Agent.Name),

		Namespace: installationNamespace,
		Args:      []string{"--wait"},
		Values:    values,
	}
}

func CreateCRDsHelmOptions(
	installationNamespace string,
	releaseName string,
	dryRun bool,
	additionalValues map[string]interface{},
) common2.HelmGenericOptions {
	values := map[string]interface{}{
		"enabled":    false,
		"installCRD": true,
	}
	maps.Copy(values, additionalValues)
	return common2.HelmGenericOptions{
		DryRun:         dryRun,
		RegistryURL:    "https://kubeshop.github.io/helm-charts",
		RepositoryName: "kubeshop",
		ChartName:      "testkube-operator",
		ReleaseName:    releaseName,

		Namespace: installationNamespace,
		Args:      []string{"--wait"},
		Values:    values,
	}
}

func PrintControlPlaneAgent(agent cloudclient.Agent) {
	agentSecretKey := agent.SecretKey
	if agent.SecretKey == "" {
		agentSecretKey = ui.LightGray("<encrypted>")
	}
	ui.Warn("ID:            ", agent.ID)
	ui.Warn("Name:          ", agent.Name)
	ui.Warn("Created:       ", agent.CreatedAt.In(time.Local).Format(time.RFC822Z)+" "+ui.LightGray("("+time.Since(agent.CreatedAt).Truncate(time.Second).String()+")"))
	if agent.DeletedAt != nil {
		ui.Warn(ui.Red("Deleted:       "), agent.DeletedAt.In(time.Local).Format(time.RFC822Z)+" "+ui.LightGray("("+time.Since(*agent.DeletedAt).Truncate(time.Second).String()+")"))
	} else if agent.Disabled {
		ui.Warn("Disabled:      ", color.Bold.Render(color.Red.Render("disabled")))
		ui.Warn("Secret Key:    ", agentSecretKey)
	} else {
		ui.Warn("Disabled:      ", "no")
		ui.Warn("Secret Key:    ", agentSecretKey)
	}
	if agent.AccessedAt == nil {
		ui.Warn("Last Activity: ", ui.LightGray("never"))
	} else {
		ui.Warn("Last Activity: ", agent.AccessedAt.In(time.Local).Format(time.RFC822Z)+" "+ui.LightGray("("+time.Since(*agent.AccessedAt).Truncate(time.Second).String()+")"))
	}

	if agent.DeletedAt != nil {
		fmt.Println("\n" + color.Bold.Render(color.Red.Render("These details are historical. The Runner has been deleted.")) + "\n")
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

	if agent.RunnerPolicy != nil && len(agent.RunnerPolicy.RequiredMatch) > 0 {
		ui.Warn("Policy:")
		ui.Warn("   Required Matching Labels:", strings.Join(agent.RunnerPolicy.RequiredMatch, ", "))
	}
}

func UiCreateAgent(cmd *cobra.Command, name string, labelPairs []string, environmentIds []string, isGlobalRunner bool, runnerGroup string, floating bool, enableRunner bool, enableListener bool) *cloudclient.Agent {
	if name == "" {
		name = ui.TextInput("agent name")
		if name == "" {
			ui.Failf("agent name is required")
		}
	}

	// Get existing agent of that name
	if existing, err := GetControlPlaneAgent(cmd, name); err == nil {
		ui.Failf("agent '%s' already exists", existing.Name)
	}

	input := cloudclient.AgentInput{
		Name:         name,
		Labels:       common.Ptr(make(map[string]string)),
		Environments: environmentIds,
		Floating:     floating,
		Type:         "run",
	}

	// Set capabilities based on resolved flags
	if enableRunner {
		input.Capabilities = append(input.Capabilities, cloudclient.AgentCapabilityRunner)
	}
	if enableListener {
		input.Capabilities = append(input.Capabilities, cloudclient.AgentCapabilityListener)
	}

	if runnerGroup != "" {
		(*input.Labels)["group"] = runnerGroup
		input.RunnerPolicy = &cloudclient.RunnerPolicy{
			RequiredMatch: []string{"group"},
		}
	} else if !isGlobalRunner {
		input.RunnerPolicy = &cloudclient.RunnerPolicy{
			RequiredMatch: []string{"name"},
		}
	}

	for _, label := range labelPairs {
		k, v, _ := strings.Cut(label, "=")
		(*input.Labels)[k] = v
	}

	envs, err := GetControlPlaneEnvironments(cmd)
	ui.ExitOnError("getting environments", err)

	if len(input.Environments) == 0 {
		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)
		envOpts := []string{envs[cfg.CloudContext.EnvironmentId].Slug}
		for id := range envs {
			if id != cfg.CloudContext.EnvironmentId {
				envOpts = append(envOpts, id)
			}
		}
		input.Environments = []string{ui.Select("select environment", envOpts)}
	}

	for i, envId := range input.Environments {
		_, ok := envs[envId]
		if !ok {
			for _, env := range envs {
				if env.Slug == envId {
					input.Environments[i] = env.Id
					break
				}
			}
		}
	}

	// Validate if the environments have the next architecture enabled
	for _, envId := range input.Environments {
		env, ok := envs[envId]
		if !ok {
			ui.Failf("unknown environment: %s", envId)
		}
		if !env.NewArchitecture {
			ui.Warn(fmt.Sprintf("Environment '%s' (%s) does not support new architecture.", env.Name, env.Id))
			if !ui.Confirm("do you want to enable it?") {
				os.Exit(1)
			}
			err := EnableNewArchitecture(cmd, env)
			ui.ExitOnError("enabling new architecture", err)
		}
	}

	agent, err := CreateAgent(cmd, input)
	ui.ExitOnError("creating agent", err)

	PrintControlPlaneAgent(*agent)

	return agent
}

func PrintKubernetesAgent(agent internalAgent) {
	ui.Warn("Pod Name:", agent.Pod.Name)
	ui.Warn("Ready:   ", fmt.Sprintf("%v", agent.Ready))
}

func GetCRDInstallation() (namespace string, releaseName string, installed bool, err error) {
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	if err != nil {
		return "", "", false, errors.Wrap(err, "getting kubernetes config")
	}
	kubeClient, err := k8sclient.ConnectToK8s()
	if err != nil {
		return "", "", false, errors.Wrap(err, "connecting to kubernetes")
	}
	internalClient, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		return "", "", false, errors.Wrap(err, "connecting to kubernetes")
	}

	list, err := internalClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", "", false, errors.Wrap(err, "discovering crds")
	}

	var crd *apiextensionsv1.CustomResourceDefinition
	for i := range list.Items {
		if list.Items[i].Spec.Group == testworkflowsv1.Group {
			crd = &list.Items[i]
			break
		}
	}
	if crd == nil {
		return "", "", false, nil
	}

	releaseName = crd.Annotations["meta.helm.sh/release-name"]
	namespace = crd.Annotations["meta.helm.sh/release-namespace"]
	if releaseName == "" || namespace == "" {
		return "", "", true, nil
	}

	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return namespace, releaseName, true, errors.Wrap(err, "getting Helm Chart release details")
	}
	var releaseSecret *corev1.Secret
	for i := range secrets.Items {
		if secrets.Items[i].Type == "helm.sh/release.v1" && secrets.Items[i].Labels["status"] == "deployed" && secrets.Items[i].Labels["name"] == releaseName {
			releaseSecret = &secrets.Items[i]
			break
		}
	}
	if releaseSecret == nil {
		return namespace, releaseName, true, fmt.Errorf("could not find details of '%s' Helm Chart release in '%s' namespace", releaseName, namespace)
	}

	releaseDataGzipped, err := base64.StdEncoding.DecodeString(string(releaseSecret.Data["release"]))
	if err != nil {
		return namespace, releaseName, true, errors.Wrapf(err, "could not decode details of '%s' Helm Chart release in '%s' namespace", releaseName, namespace)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(releaseDataGzipped))
	if err != nil {
		return namespace, releaseName, true, errors.Wrapf(err, "could not decode details of '%s' Helm Chart release in '%s' namespace", releaseName, namespace)
	}
	defer gzipReader.Close()
	releaseData, err := io.ReadAll(gzipReader)
	if err != nil {
		return namespace, releaseName, true, errors.Wrapf(err, "could not decode details of '%s' Helm Chart release in '%s' namespace", releaseName, namespace)
	}
	var data struct {
		Chart struct {
			Metadata struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"metadata"`
		} `json:"chart"`
		Config struct {
			Enabled *bool `json:"enabled,omitempty"`
		} `json:"config"`
	}
	err = json.Unmarshal(releaseData, &data)
	if err != nil {
		return namespace, releaseName, true, errors.Wrapf(err, "could not decode details of '%s' Helm Chart release in '%s' namespace", releaseName, namespace)
	}

	// Fail if the Helm Release created by CLI
	if data.Chart.Metadata.Name != "testkube-operator" || data.Config.Enabled == nil || *data.Config.Enabled {
		return namespace, releaseName, true, fmt.Errorf("CRDs are controlled by Helm Chart '%s' release in namespace '%s' that had custom installation", releaseName, namespace)
	}

	return namespace, releaseName, true, nil
}

func HasCRDsInstalled() (bool, error) {
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	if err != nil {
		return false, errors.Wrap(err, "getting kubernetes config")
	}
	internalClient, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		return false, errors.Wrap(err, "connecting to kubernetes")
	}

	list, err := internalClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "discovering crds")
	}
	for i := range list.Items {
		if list.Items[i].Spec.Group == testworkflowsv1.Group {
			return true, nil
		}
	}
	return false, nil
}
