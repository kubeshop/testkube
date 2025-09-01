package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

type HelmOptions struct {
	Name, Namespace, Chart, Values string
	NoMinio, NoMongo, NoConfirm    bool
	MinioReplicas, MongoReplicas   int
	SetOptions, ArgOptions         map[string]string

	// On-prem
	LicenseKey    string
	DemoValuesURL string

	Master config.Master
	// For debug
	DryRun         bool
	MultiNamespace bool
	NoOperator     bool
	EmbeddedNATS   bool
}

type HelmGenericOptions struct {
	DryRun      bool
	ValuesFile  string
	Args        []string
	ReleaseName string

	RegistryURL    string
	RepositoryName string
	ChartName      string

	Namespace string
	Values    map[string]interface{}
}

const (
	github                = "GitHub"
	gitlab                = "GitLab"
	dockerDaemonPrefixLen = 8
	latestReleaseUrl      = "https://api.github.com/repos/kubeshop/testkube/releases/latest"
)

func (o HelmOptions) GetApiURI() string {
	return o.Master.URIs.Api
}

func HelmUpgradeOrInstallTestkubeOnPremDemo(options HelmOptions) *CLIError {
	helmPath, cliErr := lookupHelmPath()
	if cliErr != nil {
		return cliErr
	}

	if err := updateHelmRepo(helmPath, options.DryRun, true); err != nil {
		return err
	}

	// For default setup, change the Agent host to a cluster service endpoint,
	// so it will be possible to install runners in different namespaces.
	if options.SetOptions["testkube-cloud-api.api.agent.host"] == "" && options.Namespace != "" {
		if options.SetOptions == nil {
			options.SetOptions = make(map[string]string)
		}
		options.SetOptions["testkube-cloud-api.api.agent.host"] = fmt.Sprintf("testkube-enterprise-api.%s.svc.cluster.local", options.Namespace)
	}

	// Similarly for Minio, to access by runners in different namespaces
	if options.SetOptions["testkube-cloud-api.api.minio.signing.hostname"] == "" && options.Namespace != "" {
		if options.SetOptions == nil {
			options.SetOptions = make(map[string]string)
		}
		options.SetOptions["testkube-cloud-api.api.minio.signing.hostname"] = fmt.Sprintf("testkube-enterprise-minio.%s.svc.cluster.local:9000", options.Namespace)
	}

	args := prepareTestkubeOnPremDemoArgs(options)
	output, err := runHelmCommand(helmPath, args, options.DryRun)
	if err != nil {
		return err
	}

	ui.Debug("Helm install testkube output", output)
	return nil

}

func HelmUpgradeOrInstallTestkubeAgent(options HelmOptions, cfg config.Data, isMigration bool) *CLIError {
	helmPath, cliErr := lookupHelmPath()
	if cliErr != nil {
		return cliErr
	}

	// disable mongo and minio for cloud
	options.NoMinio = true
	options.NoMongo = true

	// use config if set
	if cfg.CloudContext.AgentKey != "" && options.Master.AgentToken == "" {
		options.Master.AgentToken = cfg.CloudContext.AgentKey
	}

	if options.Master.AgentToken == "" {
		return NewCLIError(
			TKErrInvalidInstallConfig,
			"Invalid install config",
			"Provide the agent token by setting the '--agent-token' flag",
			errors.New("agent key is required"))
	}

	if cliErr := updateHelmRepo(helmPath, options.DryRun, false); cliErr != nil {
		return cliErr
	}

	args := prepareTestkubeProHelmArgs(options, isMigration)
	_, err := runHelmCommand(helmPath, args, options.DryRun)
	if err != nil {
		return err
	}

	return nil
}

func HelmUpgradeOrInstallGeneric(options HelmGenericOptions) *CLIError {
	helmPath, err := lookupHelmPath()
	if err != nil {
		return err
	}

	if err = updateHelmRepoGeneric(helmPath, options.RegistryURL, options.RepositoryName, options.DryRun); err != nil {
		return err
	}

	args := []string{
		"upgrade", "--install", "--create-namespace",
		"--namespace", options.Namespace,
	}
	if options.ValuesFile != "" {
		args = append(args, "--values", options.ValuesFile)
	}
	args = append(args, options.ReleaseName, fmt.Sprintf("%s/%s", options.RepositoryName, options.ChartName))
	for k, v := range options.Values {
		switch v.(type) {
		case int64, int32, int, uint32, uint64, bool:
			args = append(args, "--set", fmt.Sprintf("%s=%v", k, v))
		default:
			if serialized, err := json.Marshal(v); err == nil {
				args = append(args, "--set-json", fmt.Sprintf("%s=%s", k, serialized))
			} else {
				args = append(args, "--set", fmt.Sprintf("%s=%v", k, v))
			}
		}
	}
	args = append(args, options.Args...)
	output, err := runHelmCommand(helmPath, args, options.DryRun)
	if err != nil {
		return err
	}

	ui.Debug("Helm install testkube output", output)
	return nil
}

func HelmUninstall(namespace string, releaseName string) *CLIError {
	helmPath, err := lookupHelmPath()
	if err != nil {
		return err
	}
	args := []string{"uninstall", "--wait", "--namespace", namespace, releaseName}
	output, err := runHelmCommand(helmPath, args, false)
	if err != nil {
		return err
	}

	ui.Debug("Helm uninstall testkube output", output)
	return nil
}

func HelmUpgradeOrInstallTestkube(options HelmOptions) *CLIError {
	helmPath, err := lookupHelmPath()
	if err != nil {
		return err
	}

	if err = updateHelmRepo(helmPath, options.DryRun, false); err != nil {
		return err
	}

	args := prepareTestkubeHelmArgs(options)
	output, err := runHelmCommand(helmPath, args, options.DryRun)
	if err != nil {
		return err
	}

	ui.Debug("Helm install testkube output", output)
	return nil
}

func lookupHelmPath() (string, *CLIError) {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return "", NewCLIError(
			TKErrMissingDependencyHelm,
			"Required dependency not found: helm",
			"Install Helm by following this guide: https://helm.sh/docs/intro/install/",
			err,
		)
	}
	return helmPath, nil
}

func updateHelmRepoGeneric(helmPath, registryURL, registryName string, dryRun bool) *CLIError {
	_, err := runHelmCommand(helmPath, []string{"repo", "add", registryName, registryURL}, dryRun)
	errMsg := fmt.Sprintf("Error: repository name (%s) already exists, please specify a different name", registryName)
	if err != nil && !strings.Contains(err.Error(), errMsg) {
		return err
	}

	_, err = runHelmCommand(helmPath, []string{"repo", "update"}, dryRun)
	if err != nil {
		return err
	}

	return nil
}

func updateHelmRepo(helmPath string, dryRun, isOnPrem bool) *CLIError {
	if isOnPrem {
		return updateHelmRepoGeneric(helmPath, "https://kubeshop.github.io/testkube-cloud-charts", "testkubeenterprise", dryRun)
	}
	return updateHelmRepoGeneric(helmPath, "https://kubeshop.github.io/helm-charts", "kubeshop", dryRun)
}

// It cleans existing migrations job with long TTL
func CleanExistingCompletedMigrationJobs(namespace string) (cliErr *CLIError) {
	kubectlPath, cliErr := lookupKubectlPath()
	if cliErr != nil {
		return cliErr
	}

	// Clean the job only when it's found and it's state is successful - ignore pending migrations.
	cmd := []string{"get", "job", "testkube-enterprise-api-migrations", "-n", namespace, "-o", "jsonpath={.status.succeeded}"}
	succeeded, _ := runKubectlCommand(kubectlPath, cmd)
	if succeeded == "1" {
		cmd = []string{"delete", "job", "testkube-enterprise-api-migrations", "--namespace", namespace}
		_, err := runKubectlCommand(kubectlPath, cmd)
		if err != nil {
			return NewCLIError(
				TKErrCleanOldMigrationJobFailed,
				"Can't clean old migrations job",
				"Migration job can't be deleted from some reason, check for errors in installation namespace, check execution. As a workaround try to delete job manually and retry installation/upgrade process",
				err,
			).WithExecutedCommand(strings.Join(cmd, " "))
		}
	}

	return nil
}

func runHelmCommand(helmPath string, args []string, dryRun bool) (commandOutput string, cliErr *CLIError) {
	cmd := strings.Join(append([]string{helmPath}, args...), " ")
	ui.DebugNL()
	ui.Debug("Helm command:")
	ui.Debug(cmd)

	output, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	ui.DebugNL()
	ui.Debug("Helm output:")
	ui.Debug(string(output))
	if err != nil {
		return "", NewCLIError(
			TKErrHelmCommandFailed,
			"Helm command failed",
			"Retry the command with a bigger timeout by setting --helm-arg timeout=30m, if the error still persists, reach out to Testkube support",
			err,
		).WithExecutedCommand(cmd)
	}
	return string(output), nil
}

func appendHelmArgs(args []string, options HelmOptions, settings map[string]string) []string {
	for key, value := range settings {
		if _, ok := options.SetOptions[key]; !ok {
			args = append(args, "--set", fmt.Sprintf("%s=%s", key, value))
		}
	}

	for key, value := range options.SetOptions {
		args = append(args, "--set", fmt.Sprintf("%s=%s", key, value))
	}

	for key, value := range options.ArgOptions {
		args = append(args, fmt.Sprintf("--%s", key))
		if value != "" {
			args = append(args, value)
		}
	}

	return args
}

func prepareTestkubeOnPremDemoArgs(options HelmOptions) []string {
	args := []string{
		"upgrade", "--install",
		"--create-namespace",
		"--namespace", options.Namespace,
	}

	settings := map[string]string{
		"global.enterpriseLicenseKey": options.LicenseKey,
	}

	args = append(appendHelmArgs(args, options, settings), "--values", options.DemoValuesURL,
		"--wait",
		"testkube", "testkubeenterprise/testkube-enterprise")

	return args
}

// prepareTestkubeProHelmArgs prepares Helm arguments for Testkube Pro installation.
func prepareTestkubeProHelmArgs(options HelmOptions, isMigration bool) []string {
	args, settings := prepareCommonHelmArgs(options)

	settings["testkube-api.cloud.url"] = options.Master.URIs.Agent
	settings["testkube-api.cloud.key"] = options.Master.AgentToken
	settings["testkube-api.cloud.uiURL"] = options.Master.URIs.Ui
	settings["testkube-logs.pro.url"] = options.Master.URIs.Logs
	settings["testkube-logs.pro.key"] = options.Master.AgentToken

	if isMigration {
		settings["testkube-api.cloud.migrate"] = "true"
	}

	if options.Master.EnvId != "" {
		settings["testkube-api.cloud.envId"] = options.Master.EnvId
		settings["testkube-logs.pro.envId"] = options.Master.EnvId
	}

	if options.Master.OrgId != "" {
		settings["testkube-api.cloud.orgId"] = options.Master.OrgId
		settings["testkube-logs.pro.orgId"] = options.Master.OrgId
	}

	return appendHelmArgs(args, options, settings)
}

// prepareTestkubeHelmArgs prepares Helm arguments for Testkube OS installation.
func prepareTestkubeHelmArgs(options HelmOptions) []string {
	args, settings := prepareCommonHelmArgs(options)

	if options.NoMinio {
		settings["testkube-api.logs.storage"] = "mongo"
	} else {
		settings["testkube-api.logs.storage"] = "minio"
	}

	return appendHelmArgs(args, options, settings)
}

// prepareCommonHelmArgs prepares common Helm arguments for both OS and Pro installation.
func prepareCommonHelmArgs(options HelmOptions) ([]string, map[string]string) {
	args := []string{
		"upgrade", "--install", "--create-namespace",
		"--namespace", options.Namespace,
	}

	settings := map[string]string{
		"global.features.logsV2":              fmt.Sprintf("%v", options.Master.Features.LogsV2),
		"testkube-api.multinamespace.enabled": fmt.Sprintf("%t", options.MultiNamespace),
		"testkube-api.minio.enabled":          fmt.Sprintf("%t", !options.NoMinio),
		"testkube-api.minio.replicas":         fmt.Sprintf("%d", options.MinioReplicas),
		"testkube-operator.enabled":           fmt.Sprintf("%t", !options.NoOperator),
		"mongodb.enabled":                     fmt.Sprintf("%t", !options.NoMongo),
		"mongodb.replicas":                    fmt.Sprintf("%d", options.MongoReplicas),
	}

	if options.Values != "" {
		args = append(args, "--values", options.Values)
	}

	// if embedded nats is enabled disable nats chart
	if options.EmbeddedNATS {
		settings["testkube-api.nats.enabled"] = "false"
		settings["testkube-api.nats.embedded"] = "true"
	}

	args = append(args, options.Name, options.Chart)
	return args, settings
}

func PopulateHelmFlags(cmd *cobra.Command, options *HelmOptions) {
	cmd.Flags().StringVar(&options.Chart, "chart", "kubeshop/testkube", "chart name (usually you don't need to change it)")
	cmd.Flags().StringVar(&options.Name, "name", "testkube", "installation name (usually you don't need to change it)")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().StringVar(&options.Values, "values", "", "path to Helm values file")

	cmd.Flags().BoolVar(&options.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&options.NoMongo, "no-mongo", false, "don't install MongoDB")
	cmd.Flags().BoolVar(&options.NoConfirm, "no-confirm", false, "don't ask for confirmation - unatended installation mode")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "dry run mode - only print commands that would be executed")
	cmd.Flags().BoolVar(&options.EmbeddedNATS, "embedded-nats", false, "embedded NATS server in agent")
}

func PopulateLoginDataToContext(orgID, envID, token, refreshToken, dockerContainerName string, options HelmOptions, cfg config.Data) error {
	if options.Master.AgentToken != "" {
		cfg.CloudContext.AgentKey = options.Master.AgentToken
	}
	if options.Master.URIs.Agent != "" {
		cfg.CloudContext.AgentUri = options.Master.URIs.Agent
	}
	if options.Master.URIs.Ui != "" {
		cfg.CloudContext.UiUri = options.Master.URIs.Ui
	}
	if options.Master.URIs.Api != "" {
		cfg.CloudContext.ApiUri = options.Master.URIs.Api
		if options.Master.URIs.Agent == "" {
			cfg.CloudContext.AgentUri = options.Master.URIs.Api
		}
	}
	if options.Master.URIs.Auth != "" {
		cfg.CloudContext.AuthUri = options.Master.URIs.Auth
	}
	cfg.ContextType = config.ContextTypeCloud
	cfg.CloudContext.OrganizationId = orgID
	cfg.CloudContext.EnvironmentId = envID
	cfg.CloudContext.TokenType = config.TokenTypeOIDC
	if token != "" {
		cfg.CloudContext.ApiKey = token
	}
	if refreshToken != "" {
		cfg.CloudContext.RefreshToken = refreshToken
	}
	cfg.CloudContext.DockerContainerName = dockerContainerName
	if options.Master.CallbackPort != 0 {
		cfg.CloudContext.CallbackPort = options.Master.CallbackPort
	}

	cfg, err := PopulateOrgAndEnvNames(cfg, orgID, envID, options.Master.URIs.Api)
	if err != nil {
		return errors.Wrap(err, "error populating org and env names")
	}

	return config.Save(cfg)
}

func PopulateAgentDataToContext(options HelmOptions, cfg config.Data) error {
	updated := false
	if options.Master.AgentToken != "" {
		cfg.CloudContext.AgentKey = options.Master.AgentToken
		updated = true
	}
	if options.Master.URIs.Api != "" {
		cfg.CloudContext.AgentUri = options.Master.URIs.Api
		updated = true
	}
	if options.Master.URIs.Ui != "" {
		cfg.CloudContext.UiUri = options.Master.URIs.Ui
		updated = true
	}
	if options.Master.URIs.Api != "" {
		cfg.CloudContext.ApiUri = options.Master.URIs.Api
		updated = true
	}
	if options.Master.URIs.Auth != "" {
		cfg.CloudContext.AuthUri = options.Master.URIs.Auth
		updated = true
	}
	if options.Master.IdToken != "" {
		cfg.CloudContext.ApiKey = options.Master.IdToken
		updated = true
	}
	if options.Master.EnvId != "" {
		cfg.CloudContext.EnvironmentId = options.Master.EnvId
		updated = true
	}
	if options.Master.OrgId != "" {
		cfg.CloudContext.OrganizationId = options.Master.OrgId
		updated = true
	}
	if options.Master.CallbackPort != 0 {
		cfg.CloudContext.CallbackPort = options.Master.CallbackPort
		updated = true
	}

	if updated {
		return config.Save(cfg)
	}

	return nil
}

func IsUserLoggedIn(cfg config.Data, options HelmOptions) bool {
	if options.Master.URIs.Api != cfg.CloudContext.ApiUri {
		//different environment
		return false
	}

	if cfg.CloudContext.ApiKey != "" && cfg.CloudContext.RefreshToken != "" {
		// users with refresh token don't need to login again
		// since on expired token they will be logged in automatically
		return true
	}
	return false
}

func UpdateTokens(cfg config.Data, token, refreshToken string) error {
	var updated bool
	if token != cfg.CloudContext.ApiKey {
		cfg.CloudContext.ApiKey = token
		updated = true
	}
	if refreshToken != cfg.CloudContext.RefreshToken {
		cfg.CloudContext.RefreshToken = refreshToken
		updated = true
	}

	if updated {
		return config.Save(cfg)
	}

	return nil
}

func PopulateOrgAndEnvNames(cfg config.Data, orgId, envId, apiUrl string) (config.Data, error) {
	if orgId != "" {
		cfg.CloudContext.OrganizationId = orgId
		// reset env when the org is changed
		if envId == "" {
			cfg.CloudContext.EnvironmentId = ""
		}
	}
	if envId != "" {
		cfg.CloudContext.EnvironmentId = envId
	}

	orgClient := cloudclient.NewOrganizationsClient(apiUrl, cfg.CloudContext.ApiKey)
	org, err := orgClient.Get(cfg.CloudContext.OrganizationId)
	if err != nil {
		return cfg, errors.Wrap(err, "error getting organization")
	}

	envsClient := cloudclient.NewEnvironmentsClient(apiUrl, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId)
	env, err := envsClient.Get(cfg.CloudContext.EnvironmentId)
	if err != nil {
		return cfg, errors.Wrap(err, "error getting environment")
	}

	cfg.CloudContext.OrganizationName = org.Name
	cfg.CloudContext.EnvironmentName = env.Name

	return cfg, nil
}

func PopulateCloudConfig(cfg config.Data, apiKey string, dockerContainerName *string, opts *HelmOptions) config.Data {
	if apiKey != "" {
		cfg.CloudContext.ApiKey = apiKey
	}

	cfg.CloudContext.ApiUri = opts.Master.URIs.Api
	cfg.CloudContext.UiUri = opts.Master.URIs.Ui
	cfg.CloudContext.AgentUri = opts.Master.URIs.Agent
	if dockerContainerName != nil {
		cfg.CloudContext.DockerContainerName = *dockerContainerName
	}
	cfg.CloudContext.CallbackPort = opts.Master.CallbackPort

	return cfg
}

func LoginUser(authUri string, customConnector bool, port int) (string, string, error) {
	ui.H1("Login")
	connectorID := ""
	if !customConnector {
		connectorID = ui.Select("Choose your login method", []string{github, gitlab})
	}

	// Handle the common case where th Demo instance is running on reserved port

	ui.Debug("Logging into cloud with parameters", authUri, connectorID)
	authUrl, tokenChan, err := cloudlogin.CloudLogin(context.Background(), authUri, strings.ToLower(connectorID), port)
	if err != nil {
		return "", "", fmt.Errorf("cloud login: %w", err)
	}

	ui.Paragraph("Your browser should open automatically. If not, please open this link in your browser:")
	ui.Link(authUrl)
	ui.Paragraph("(just login and get back to your terminal)")
	ui.Paragraph("")

	if ok := ui.Confirm("Continue"); !ok {
		return "", "", fmt.Errorf("login cancelled")
	}

	ui.Debug("Opening login page in browser to get a token", authUrl)
	// open browser with login page and redirect to localhost
	open.Run(authUrl)

	idToken, refreshToken, err := uiGetToken(tokenChan)
	if err != nil {
		return "", "", fmt.Errorf("getting token")
	}
	return idToken, refreshToken, nil
}

func uiGetToken(tokenChan chan cloudlogin.Tokens) (string, string, error) {
	// wait for token received to browser
	s := ui.NewSpinner("waiting for auth token")

	var token cloudlogin.Tokens
	select {
	case token = <-tokenChan:
		s.Success()
	case <-time.After(5 * time.Minute):
		s.Fail("Timeout waiting for auth token")
		return "", "", fmt.Errorf("timeout waiting for auth token")
	}
	ui.NL()

	return token.IDToken, token.RefreshToken, nil
}

func GetCurrentKubernetesContext() (string, *CLIError) {
	kubectlPath, cliErr := lookupKubectlPath()
	if cliErr != nil {
		return "", cliErr
	}

	output, err := runKubectlCommand(kubectlPath, []string{"config", "current-context"})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func KubectlScaleDeployment(namespace, deployment string, replicas int) (string, error) {
	kubectl, cliErr := lookupKubectlPath()
	if cliErr != nil {
		return "", cliErr
	}

	// kubectl patch --namespace=$n deployment $1 -p "{\"spec\":{\"replicas\": $2}}"
	out, err := process.Execute(kubectl, "patch", "--namespace", namespace, "deployment", deployment, "-p", fmt.Sprintf("{\"spec\":{\"replicas\": %d}}", replicas))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func KubectlLogs(namespace string, labels map[string]string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"logs",
		"--all-containers",
		"-n", namespace,
		"--max-log-requests=100",
		"--tail=10000000",
		"--ignore-errors=true",
	}

	for k, v := range labels {
		args = append(args, fmt.Sprintf("-l %s=%s", k, v))
	}

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlPrintEvents(namespace string) error {
	kubectl, cliErr := lookupKubectlPath()
	if cliErr != nil {
		return cliErr
	}

	args := []string{
		"get",
		"events",
		"-n", namespace,
	}

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	err := process.ExecuteAndStreamOutput(kubectl, args...)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"events",
		"-A",
	}

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlVersion() (client string, server string, err error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", "", err
	}

	args := []string{
		"version",
		"-o", "json",
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	out, eerr := process.Execute(kubectl, args...)
	if eerr != nil {
		return "", "", eerr
	}

	type Version struct {
		ClientVersion struct {
			Version string `json:"gitVersion,omitempty"`
		} `json:"clientVersion,omitempty"`
		ServerVersion struct {
			Version string `json:"gitVersion,omitempty"`
		} `json:"serverVersion,omitempty"`
	}

	var v Version

	out, err = extractJSONObject(out)
	if err != nil {
		return "", "", err
	}

	err = json.Unmarshal(out, &v)
	if err != nil {
		return "", "", err
	}

	return strings.TrimLeft(v.ClientVersion.Version, "v"), strings.TrimLeft(v.ServerVersion.Version, "v"), nil
}

func KubectlDescribePods(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"describe",
		"pods",
		"-n", namespace,
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlPrintPods(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"pods",
		"-n", namespace,
		"--show-labels",
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlGetStorageClass(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"storageclass",
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlGetServices(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"services",
		"-n", namespace,
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlDescribeServices(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"services",
		"-n", namespace,
		"-o", "yaml",
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlGetIngresses(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"ingresses",
		"-n", namespace,
	}

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlDescribeIngresses(namespace string) error {
	kubectl, err := lookupKubectlPath()
	if err != nil {
		return err
	}

	args := []string{
		"get",
		"ingresses",
		"-n", namespace,
		"-o", "yaml",
	}

	ui.ShellCommand(kubectl, args...)
	ui.NL()

	return process.ExecuteAndStreamOutput(kubectl, args...)
}

func KubectlGetNamespacesHavingSecrets(secretName string) ([]string, error) {
	kubectl, clierr := lookupKubectlPath()
	if clierr != nil {
		return nil, clierr.ActualError
	}

	args := []string{
		"get",
		"secret",
		"-A",
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	out, err := process.Execute(kubectl, args...)
	if err != nil {
		return nil, err
	}

	nss := extractUniqueNamespaces(string(out), secretName)
	return nss, nil
}

func extractUniqueNamespaces(data string, secretName string) []string {
	// Split the data into lines
	lines := strings.Split(data, "\n")

	// Map to store unique namespaces
	uniq := make(map[string]bool)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		if parts[1] == secretName {
			uniq[parts[0]] = true
		}
	}

	// Convert map keys (namespaces) to a slice of strings
	list := make([]string, 0, len(uniq))
	for namespace := range uniq {
		list = append(list, namespace)
	}

	return list
}

func KubectlGetPodEnvs(selector, namespace string) (map[string]string, error) {
	kubectl, clierr := lookupKubectlPath()
	if clierr != nil {
		return nil, clierr.ActualError
	}

	args := []string{
		"get",
		"secret",
		selector,
		"-n", namespace,
		"-o", `jsonpath='{range .items[*].spec.containers[*]}{"\nContainer: "}{.name}{"\n"}{range .env[*]}{.name}={.value}{"\n"}{end}{end}'`,
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	out, err := process.Execute(kubectl, args...)
	if err != nil {
		return nil, err
	}

	return convertEnvToMap(string(out)), nil
}

func KubectlGetSecret(selector, namespace string) (map[string]string, error) {
	kubectl, clierr := lookupKubectlPath()
	if clierr != nil {
		return nil, clierr.ActualError
	}

	args := []string{
		"get",
		"secret",
		selector,
		"-n", namespace,
		"-o", `jsonpath='{.data}'`,
	}

	if ui.IsVerbose() {
		ui.ShellCommand(kubectl, args...)
		ui.NL()
	}

	out, err := process.Execute(kubectl, args...)
	if err != nil {
		return nil, err
	}

	return secretsJSONToMap(string(out))
}

func lookupKubectlPath() (string, *CLIError) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return "", NewCLIError(
			TKErrMissingDependencyKubectl,
			"Required dependency not found: kubectl",
			"Install kubectl by following this guide: https://kubernetes.io/docs/tasks/tools/#kubectl",
			err,
		)
	}
	return kubectlPath, nil
}

func runKubectlCommand(kubectlPath string, args []string) (output string, cliErr *CLIError) {
	cmd := strings.Join(append([]string{kubectlPath}, args...), " ")
	ui.DebugNL()
	ui.Debug("Kubectl command:")
	ui.Debug(cmd)
	out, err := process.Execute(kubectlPath, args...)
	ui.DebugNL()
	ui.Debug("Kubectl output:")
	ui.Debug(string(out))
	if err != nil {
		return "", NewCLIError(
			TKErrKubectlCommandFailed,
			"Kubectl command failed",
			"Check does the kubeconfig file (~/.kube/config) exist and has correct permissions and is the Kubernetes cluster reachable and has Ready nodes by running 'kubectl get nodes' ",
			err,
		).WithExecutedCommand(cmd)
	}
	return string(out), nil
}

func UiGetNamespace(cmd *cobra.Command, defaultNamespace string) string {
	var namespace string
	var err error

	if cmd.Flag("namespace").Changed {
		namespace, err = cmd.Flags().GetString("namespace")
		ui.ExitOnError("getting namespace", err)
	} else {
		namespace = ui.TextInput("Please provide namespace for Control Plane", defaultNamespace)
	}

	return namespace
}

func RunDockerCommand(args []string) (output string, cliErr *CLIError) {
	out, err := process.Execute("docker", args...)
	if err != nil {
		return "", NewCLIError(
			TKErrDockerCommandFailed,
			"Docker command failed",
			"Check is the Docker service installed and running on your computer by executing 'docker info' ",
			err,
		).WithExecutedCommand(strings.Join(append([]string{"docker"}, args...), " "))
	}
	return string(out), nil
}

func DockerRunTestkubeAgent(options HelmOptions, cfg config.Data, dockerContainerName, dockerImage string) *CLIError {
	// use config if set
	if cfg.CloudContext.AgentKey != "" && options.Master.AgentToken == "" {
		options.Master.AgentToken = cfg.CloudContext.AgentKey
	}

	if options.Master.AgentToken == "" {
		return NewCLIError(
			TKErrInvalidInstallConfig,
			"Invalid install config",
			"Provide the agent token by setting the '--agent-token' flag",
			errors.New("agent key is required"))
	}

	args := prepareTestkubeProDockerArgs(options, dockerContainerName, dockerImage)
	output, err := RunDockerCommand(args)
	if err != nil {
		return err
	}

	ui.Debug("Docker command output:")
	ui.Debug("Arguments", args...)

	ui.Debug("Docker run testkube output", output)

	return nil
}

// prepareTestkubeProDockerArgs prepares docker arguments for Testkube Pro running.
func prepareTestkubeProDockerArgs(options HelmOptions, dockerContainerName, dockerImage string) []string {
	args := []string{
		"run",
		"--name", dockerContainerName,
		"--privileged",
		"-d",
		"-e", "CLOUD_URL=" + options.Master.URIs.Agent,
		"-e", "AGENT_KEY=" + options.Master.AgentToken,
		dockerImage,
	}

	return args
}

// prepareTestkubeUpgradeDockerArgs prepares docker arguments for Testkube Upgrade running.
func prepareTestkubeUpgradeDockerArgs(options HelmOptions, dockerContainerName, latestVersion string) []string {
	args := []string{
		"exec",
		dockerContainerName,
		"helm",
		"upgrade",
		// These arguments are similar to Docker entrypoint script
		"testkube",
		"testkube/testkube",
		"--namespace",
		"testkube",
		"--set",
		"testkube-api.minio.enabled=false",
		"--set",
		"mongodb.enabled=false",
		"--set",
		"testkube-api.cloud.key=" + options.Master.AgentToken,
		"--set",
		"testkube-api.cloud.url=" + options.Master.URIs.Agent,
		"--set",
		"testkube-api.dockerImageVersion=" + latestVersion,
	}

	return args
}

func StreamDockerLogs(dockerContainerName string) *CLIError {
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return NewCLIError(
			TKErrInvalidDockerConfig,
			"Invalid docker config",
			"Check your environment variables used to connect to Docker daemon",
			err)
	}

	ctx := context.Background()
	// Set options to stream logs and show both stdout and stderr logs
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Follow logs in real-time
		Timestamps: false,
	}

	// Fetch logs from the container
	logs, err := cli.ContainerLogs(ctx, dockerContainerName, opts)
	if err != nil {
		return NewCLIError(
			TKErrDockerLogStreamingFailed,
			"Docker log streaming failed",
			"Check that your Testkube Docker Agent container is up and runnning",
			err)
	}
	defer logs.Close()

	// Use a buffered scanner to read the logs line by line
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > dockerDaemonPrefixLen {
			line = line[dockerDaemonPrefixLen:]
		}

		if ui.IsVerbose() {
			fmt.Println(string(line)) // Optional: print logs to console
		}

		if strings.Contains(string(line), "Testkube installation succeed") {
			break
		}

		if strings.Contains(string(line), "Testkube installation failed") {
			return NewCLIError(
				TKErrDockerInstallationFailed,
				"Docker installation failed",
				"Check logs of your Testkube Docker Agent container",
				errors.New(string(line)))
		}
	}

	if err := scanner.Err(); err != nil {
		return NewCLIError(
			TKErrDockerLogReadingFailed,
			"Docker log reading failed",
			"Check logs of your Testkube Docker Agent container",
			err)
	}

	return nil
}

func DockerUpgradeTestkubeAgent(options HelmOptions, latestVersion string, cfg config.Data) *CLIError {
	// use config if set
	if cfg.CloudContext.AgentKey != "" && options.Master.AgentToken == "" {
		options.Master.AgentToken = cfg.CloudContext.AgentKey
	}

	if options.Master.AgentToken == "" {
		return NewCLIError(
			TKErrInvalidInstallConfig,
			"Invalid install config",
			"Provide the agent token by setting the '--agent-token' flag",
			errors.New("agent key is required"))
	}

	args := prepareTestkubeUpgradeDockerArgs(options, cfg.CloudContext.DockerContainerName, latestVersion)
	output, err := RunDockerCommand(args)
	if err != nil {
		return err
	}

	ui.Debug("Docker command output:")
	ui.Debug("Arguments", args...)

	ui.Debug("Docker run testkube output", output)

	return nil
}

type releaseMetadata struct {
	TagName string `json:"tag_name"`
}

func GetLatestVersion() (string, error) {
	resp, err := http.Get(latestReleaseUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var metadata releaseMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return "", err
	}

	return strings.TrimPrefix(metadata.TagName, "v"), nil
}

func convertEnvToMap(input string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Split on first = only
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Store in map
		result[key] = value
	}

	return result
}

func secretsJSONToMap(in string) (map[string]string, error) {
	res := map[string]string{}
	in = strings.TrimLeft(in, "'")
	in = strings.TrimRight(in, "'")
	err := json.Unmarshal([]byte(in), &res)

	if len(res) > 0 {
		for k := range res {
			decoded, err := base64.StdEncoding.DecodeString(res[k])
			if err != nil {
				return nil, err
			}
			res[k] = string(decoded)
		}
	}

	return res, err
}

// extractJSONObject extracts JSON from any string
func extractJSONObject(input []byte) ([]byte, error) {
	// Find the first '{' and last '}' to extract JSON object
	start := bytes.Index(input, []byte("{"))
	end := bytes.LastIndex(input, []byte("}"))

	if start == -1 || end == -1 || start > end {
		return []byte(""), fmt.Errorf("invalid JSON format")
	}

	jsonStr := input[start : end+1]

	// Validate JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  "); err != nil {
		return []byte(""), err
	}

	return prettyJSON.Bytes(), nil
}

// CheckLegacyName checks if the given resource type is legacy and shows a deprecation warning.
// This function should be called in PersistentPreRun functions of commands that operate on legacy resources.
// Legacy resource types include: "test", "testsuite", "executor", "testsource".
// Usage: common.CheckLegacyName(cmd.Name())
func CheckLegacyName(commandName string) {
	// Legacy resource types that are about to be deprecated
	legacyCommandNames := map[string]bool{
		"test":                true,
		"testsuite":           true,
		"executor":            true,
		"testsource":          true,
		"execution":           true,
		"executions":          true,
		"testsuiteexecution":  true,
		"testsuiteexecutions": true,
	}

	if legacyCommandNames[commandName] {
		ui.Info("This functionality is about to be deprecated, read more at https://docs.testkube.io/articles/legacy-features")
	}
}
