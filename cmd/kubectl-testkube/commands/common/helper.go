package common

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/migrations"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

type HelmOptions struct {
	Name, Namespace, Chart, Values string
	NoMinio, NoMongo, NoConfirm    bool
	MinioReplicas, MongoReplicas   int

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

const (
	github = "GitHub"
	gitlab = "GitLab"
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
	output, err := runHelmCommand(helmPath, args, options.DryRun)
	if err != nil {
		return err
	}

	ui.Debug("Helm command output:")
	ui.Debug(helmPath, args...)

	ui.Debug("Helm install testkube output", output)

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

func updateHelmRepo(helmPath string, dryRun bool, isOnPrem bool) *CLIError {
	registryURL := "https://kubeshop.github.io/helm-charts"
	registryName := "kubeshop"
	if isOnPrem {
		registryURL = "https://kubeshop.github.io/testkube-cloud-charts"
		registryName = "testkubeenterprise"
	}
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

func runHelmCommand(helmPath string, args []string, dryRun bool) (commandOutput string, cliErr *CLIError) {
	output, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: dryRun})
	if err != nil {
		return "", NewCLIError(
			TKErrHelmCommandFailed,
			"Helm command failed",
			"Retry the command with a bigger timeout by setting --timeout 30m, if the error still persists, reach out to Testkube support",
			err,
		)
	}
	return string(output), nil
}

func prepareTestkubeOnPremDemoArgs(options HelmOptions) []string {
	return []string{
		"upgrade", "--install",
		"--create-namespace",
		"--namespace", options.Namespace,
		"--set", "global.enterpriseLicenseKey=" + options.LicenseKey,
		"--values", options.DemoValuesURL,
		"--wait",
		"testkube", "testkubeenterprise/testkube-enterprise"}
}

// prepareTestkubeProHelmArgs prepares Helm arguments for Testkube Pro installation.
func prepareTestkubeProHelmArgs(options HelmOptions, isMigration bool) []string {
	args := prepareCommonHelmArgs(options)

	args = append(args,
		"--set", "testkube-api.cloud.url="+options.Master.URIs.Agent,
		"--set", "testkube-api.cloud.key="+options.Master.AgentToken,
		"--set", "testkube-api.cloud.uiURL="+options.Master.URIs.Ui,
		"--set", "testkube-logs.pro.url="+options.Master.URIs.Logs,
		"--set", "testkube-logs.pro.key="+options.Master.AgentToken,
	)

	if isMigration {
		args = append(args, "--set", "testkube-api.cloud.migrate=true")
	}

	if options.Master.EnvId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.envId=%s", options.Master.EnvId))
		args = append(args, "--set", fmt.Sprintf("testkube-logs.pro.envId=%s", options.Master.EnvId))
	}

	if options.Master.RunnerId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.runnerId=%s", options.Master.RunnerId))
	}

	if options.Master.OrgId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.orgId=%s", options.Master.OrgId))
		args = append(args, "--set", fmt.Sprintf("testkube-logs.pro.orgId=%s", options.Master.OrgId))
	}

	return args
}

// prepareTestkubeHelmArgs prepares Helm arguments for Testkube OS installation.
func prepareTestkubeHelmArgs(options HelmOptions) []string {
	args := prepareCommonHelmArgs(options)

	if options.NoMinio {
		args = append(args, "--set", "testkube-api.logs.storage=mongo")
	} else {
		args = append(args, "--set", "testkube-api.logs.storage=minio")
	}

	return args
}

// prepareCommonHelmArgs prepares common Helm arguments for both OS and Pro installation.
func prepareCommonHelmArgs(options HelmOptions) []string {
	args := []string{
		"upgrade", "--install", "--create-namespace",
		"--namespace", options.Namespace,
		"--set", fmt.Sprintf("global.features.logsV2=%v", options.Master.Features.LogsV2),
		"--set", fmt.Sprintf("testkube-api.multinamespace.enabled=%t", options.MultiNamespace),
		"--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio),
		"--set", fmt.Sprintf("testkube-api.minio.replicas=%d", options.MinioReplicas), "--set", fmt.Sprintf("testkube-operator.enabled=%t", !options.NoOperator),
		"--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo),
		"--set", fmt.Sprintf("mongodb.replicas=%d", options.MongoReplicas),
	}

	if options.Values != "" {
		args = append(args, "--values", options.Values)
	}

	// if embedded nats is enabled disable nats chart
	if options.EmbeddedNATS {
		args = append(args, "--set", "testkube-api.nats.enabled=false")
		args = append(args, "--set", "testkube-api.nats.embedded=true")
	}

	args = append(args, options.Name, options.Chart)
	return args
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

func PopulateLoginDataToContext(orgID, envID, token, refreshToken string, options HelmOptions, cfg config.Data) error {
	if options.Master.AgentToken != "" {
		cfg.CloudContext.AgentKey = options.Master.AgentToken
	}
	if options.Master.URIs.Api != "" {
		cfg.CloudContext.AgentUri = options.Master.URIs.Api
	}
	if options.Master.URIs.Ui != "" {
		cfg.CloudContext.UiUri = options.Master.URIs.Ui
	}
	if options.Master.URIs.Api != "" {
		cfg.CloudContext.ApiUri = options.Master.URIs.Api
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

func RunAgentMigrations(cmd *cobra.Command) (hasMigrations bool, err error) {
	client, _, err := GetClient(cmd)
	ui.ExitOnError("getting client", err)

	info, err := client.GetServerInfo()
	ui.ExitOnError("getting server info", err)

	if info.Version == "" {
		ui.Failf("Can't detect cluster version")
	}

	ui.Info("Available agent migrations for", info.Version)
	results := migrations.Migrator.GetValidMigrations(info.Version, migrator.MigrationTypeClient)
	if len(results) == 0 {
		ui.Warn("No agent migrations available for", info.Version)
		return false, nil
	}

	for _, migration := range results {
		fmt.Printf("- %+v - %s\n", migration.Version(), migration.Info())
	}

	return true, migrations.Migrator.Run(info.Version, migrator.MigrationTypeClient)
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

func PopulateCloudConfig(cfg config.Data, apiKey string, opts *HelmOptions) config.Data {
	if apiKey != "" {
		cfg.CloudContext.ApiKey = apiKey
	}

	cfg.CloudContext.ApiUri = opts.Master.URIs.Api
	cfg.CloudContext.UiUri = opts.Master.URIs.Ui
	cfg.CloudContext.AgentUri = opts.Master.URIs.Agent

	return cfg
}

func LoginUser(authUri string) (string, string, error) {
	ui.H1("Login")
	connectorID := ui.Select("Choose your login method", []string{github, gitlab})

	authUrl, tokenChan, err := cloudlogin.CloudLogin(context.Background(), authUri, strings.ToLower(connectorID))
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

	ui.ShellCommand(kubectl, args...)
	ui.NL()

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

	ui.ShellCommand(kubectl, args...)
	ui.NL()

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

	ui.ShellCommand(kubectl, args...)
	ui.NL()

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

	ui.ShellCommand(kubectl, args...)
	ui.NL()

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

	ui.ShellCommand(kubectl, args...)
	ui.NL()

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
	out, err := process.Execute(kubectlPath, args...)
	if err != nil {
		return "", NewCLIError(
			TKErrKubectlCommandFailed,
			"Kubectl command failed",
			"Check does the kubeconfig file (~/.kube/config) exist and has correct permissions and is the Kubernetes cluster reachable and has Ready nodes by running 'kubectl get nodes' ",
			err,
		)
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
