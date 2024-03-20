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

	Master config.Master
	// For debug
	DryRun         bool
	MultiNamespace bool
	NoOperator     bool
}

const (
	github = "GitHub"
	gitlab = "GitLab"
)

func (o HelmOptions) GetApiURI() string {
	return o.Master.URIs.Api
}

func GetCurrentKubernetesContext() (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", err
	}

	out, err := process.Execute(kubectl, "config", "current-context")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func HelmUpgradeOrInstallTestkubeCloud(options HelmOptions, cfg config.Data, isMigration bool) error {
	// use config if set
	if cfg.CloudContext.AgentKey != "" && options.Master.AgentToken == "" {
		options.Master.AgentToken = cfg.CloudContext.AgentKey
	}

	if options.Master.AgentToken == "" {
		return fmt.Errorf("agent key and agent uri are required, please pass it with `--agent-token` and `--agent-uri` flags")
	}

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return err
	}

	// repo update
	args := []string{"repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts"}
	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: options.DryRun})
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
		ui.WarnOnError("adding testkube repo", err)
	}

	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: []string{"repo", "update"}, DryRun: options.DryRun})
	ui.ExitOnError("updating helm repositories", err)

	// upgrade cloud
	args = []string{
		"upgrade", "--install", "--create-namespace",
		"--namespace", options.Namespace,
		"--set", "testkube-api.cloud.url=" + options.Master.URIs.Agent,
		"--set", "testkube-api.cloud.key=" + options.Master.AgentToken,
		"--set", "testkube-api.cloud.uiURL=" + options.Master.URIs.Ui,
		"--set", "testkube-logs.pro.url=" + options.Master.URIs.Logs,
		"--set", "testkube-logs.pro.key=" + options.Master.AgentToken,
	}
	if isMigration {
		args = append(args, "--set", "testkube-api.cloud.migrate=true")
	}

	if options.Master.EnvId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.envId=%s", options.Master.EnvId))
		args = append(args, "--set", fmt.Sprintf("testkube-logs.pro.envId=%s", options.Master.EnvId))
	}
	if options.Master.OrgId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.orgId=%s", options.Master.OrgId))
		args = append(args, "--set", fmt.Sprintf("testkube-logs.pro.orgId=%s", options.Master.OrgId))
	}

	args = append(args, "--set", fmt.Sprintf("global.features.logsV2=%v", options.Master.Features.LogsV2))

	args = append(args, "--set", fmt.Sprintf("testkube-api.multinamespace.enabled=%t", options.MultiNamespace))
	args = append(args, "--set", fmt.Sprintf("testkube-operator.enabled=%t", !options.NoOperator))
	args = append(args, "--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo))
	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio))

	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.replicas=%d", options.MinioReplicas))
	args = append(args, "--set", fmt.Sprintf("mongodb.replicas=%d", options.MongoReplicas))

	args = append(args, options.Name, options.Chart)

	if options.Values != "" {
		args = append(args, "--values", options.Values)
	}

	out, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: options.DryRun})
	if err != nil {
		return err
	}

	ui.Debug("Helm command output:")
	ui.Debug(helmPath, args...)

	ui.Debug("Helm install testkube output", string(out))

	return nil
}

func HelmUpgradeOrInstalTestkube(options HelmOptions) error {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return err
	}

	ui.Info("Helm installing testkube framework")
	args := []string{"repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts"}
	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: options.DryRun})
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
		ui.WarnOnError("adding testkube repo", err)
	}

	_, err = process.ExecuteWithOptions(process.Options{Command: helmPath, Args: []string{"repo", "update"}, DryRun: options.DryRun})
	ui.ExitOnError("updating helm repositories", err)

	args = []string{"upgrade", "--install", "--create-namespace", "--namespace", options.Namespace}
	args = append(args, "--set", fmt.Sprintf("testkube-api.multinamespace.enabled=%t", options.MultiNamespace))
	args = append(args, "--set", fmt.Sprintf("testkube-operator.enabled=%t", !options.NoOperator))
	args = append(args, "--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo))
	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio))
	if options.NoMinio {
		args = append(args, "--set", "testkube-api.logs.storage=mongo")
	} else {
		args = append(args, "--set", "testkube-api.logs.storage=minio")
	}

	args = append(args, "--set", fmt.Sprintf("global.features.logsV2=%v", options.Master.Features.LogsV2))

	args = append(args, options.Name, options.Chart)

	if options.Values != "" {
		args = append(args, "--values", options.Values)
	}

	out, err := process.ExecuteWithOptions(process.Options{Command: helmPath, Args: args, DryRun: options.DryRun})
	if err != nil {
		return err
	}

	ui.Debug("Helm install testkube output", string(out))
	return nil
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

func KubectlScaleDeployment(namespace, deployment string, replicas int) (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", err
	}

	// kubectl patch --namespace=$n deployment $1 -p "{\"spec\":{\"replicas\": $2}}"
	out, err := process.Execute(kubectl, "patch", "--namespace", namespace, "deployment", deployment, "-p", fmt.Sprintf("{\"spec\":{\"replicas\": %d}}", replicas))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func RunMigrations(cmd *cobra.Command) (hasMigrations bool, err error) {
	client, _, err := GetClient(cmd)
	ui.ExitOnError("getting client", err)

	info, err := client.GetServerInfo()
	ui.ExitOnError("getting server info", err)

	if info.Version == "" {
		ui.Failf("Can't detect cluster version")
	}

	ui.Info("Available migrations for", info.Version)
	results := migrations.Migrator.GetValidMigrations(info.Version, migrator.MigrationTypeClient)
	if len(results) == 0 {
		ui.Warn("No migrations available for", info.Version)
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

	var err error
	cfg, err = PopulateOrgAndEnvNames(cfg, opts.Master.OrgId, opts.Master.EnvId, opts.Master.URIs.Api)
	if err != nil {
		ui.Failf("Error populating org and env names: %s", err)
	}

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
