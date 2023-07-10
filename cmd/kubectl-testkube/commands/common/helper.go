package common

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/migrations"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

type HelmOptions struct {
	Name, Namespace, Chart, Values                  string
	NoDashboard, NoMinio, NoMongo, NoConfirm        bool
	MinioReplicas, MongoReplicas, DashboardReplicas int
	// Cloud only params
	CloudAgentToken        string
	CloudIdToken           string
	CloudRootDomain        string
	CloudOrgId, CloudEnvId string
	CloudUris              CloudUris
	// For debug
	DryRun bool
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
	if cfg.CloudContext.AgentKey != "" && options.CloudAgentToken == "" {
		options.CloudAgentToken = cfg.CloudContext.AgentKey
	}

	if cfg.CloudContext.RootDomain != "" && options.CloudRootDomain == "" {
		options.CloudUris = NewCloudUris(cfg.CloudContext.RootDomain)
	}

	if options.CloudAgentToken == "" {
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
		"--set", "testkube-api.cloud.url=" + options.CloudUris.Agent,
		"--set", "testkube-api.cloud.key=" + options.CloudAgentToken,
	}
	if isMigration {
		args = append(args, "--set", "testkube-api.cloud.migrate=true")
	}

	if options.CloudEnvId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.envId=%s", options.CloudEnvId))
	}
	if options.CloudOrgId != "" {
		args = append(args, "--set", fmt.Sprintf("testkube-api.cloud.orgId=%s", options.CloudOrgId))
	}

	args = append(args, "--set", fmt.Sprintf("testkube-dashboard.enabled=%t", !options.NoDashboard))
	args = append(args, "--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo))
	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio))

	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.replicas=%d", options.MinioReplicas))
	args = append(args, "--set", fmt.Sprintf("mongodb.replicas=%d", options.MongoReplicas))
	args = append(args, "--set", fmt.Sprintf("testkube-dashboard.replicas=%d", options.DashboardReplicas))

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
	args = append(args, "--set", fmt.Sprintf("testkube-dashboard.enabled=%t", !options.NoDashboard))
	args = append(args, "--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo))
	args = append(args, "--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio))
	if options.NoMinio {
		args = append(args, "--set", "testkube-api.logs.storage=mongo")
	} else {
		args = append(args, "--set", "testkube-api.logs.storage=minio")
	}

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

	cmd.Flags().StringVar(&options.CloudUris.Agent, "agent-uri", "", "Testkube Cloud agent URI [required for cloud mode]")
	cmd.Flags().StringVar(&options.CloudAgentToken, "agent-token", "", "Testkube Cloud agent key [required for cloud mode]")
	cmd.Flags().StringVar(&options.CloudOrgId, "org-id", "", "Testkube Cloud organization id [required for cloud mode]")
	cmd.Flags().StringVar(&options.CloudEnvId, "env-id", "", "Testkube Cloud environment id [required for cloud mode]")

	cmd.Flags().StringVar(&options.CloudRootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for cloud mode]")

	cmd.Flags().BoolVar(&options.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&options.NoDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&options.NoMongo, "no-mongo", false, "don't install MongoDB")
	cmd.Flags().BoolVar(&options.NoConfirm, "no-confirm", false, "don't ask for confirmation - unatended installation mode")

	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "dry run mode - only print commands that would be executed")
}

func PopulateLoginDataToContext(orgID, envID, token, refreshToken string, options HelmOptions, cfg config.Data) error {
	if options.CloudAgentToken != "" {
		cfg.CloudContext.AgentKey = options.CloudAgentToken
	}
	if options.CloudUris.Api != "" {
		cfg.CloudContext.AgentUri = options.CloudUris.Api
	}
	if options.CloudUris.Ui != "" {
		cfg.CloudContext.UiUri = options.CloudUris.Ui
	}
	if options.CloudUris.Api != "" {
		cfg.CloudContext.ApiUri = options.CloudUris.Api
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

	cfg, err := PopulateOrgAndEnvNames(cfg, orgID, envID, options.CloudRootDomain)
	if err != nil {
		return err
	}

	return config.Save(cfg)
}

func PopulateAgentDataToContext(options HelmOptions, cfg config.Data) error {
	updated := false
	if options.CloudAgentToken != "" {
		cfg.CloudContext.AgentKey = options.CloudAgentToken
		updated = true
	}
	if options.CloudUris.Api != "" {
		cfg.CloudContext.AgentUri = options.CloudUris.Api
		updated = true
	}
	if options.CloudUris.Ui != "" {
		cfg.CloudContext.UiUri = options.CloudUris.Ui
		updated = true
	}
	if options.CloudUris.Api != "" {
		cfg.CloudContext.ApiUri = options.CloudUris.Api
		updated = true
	}
	if options.CloudIdToken != "" {
		cfg.CloudContext.ApiKey = options.CloudIdToken
		updated = true
	}
	if options.CloudEnvId != "" {
		cfg.CloudContext.EnvironmentId = options.CloudEnvId
		updated = true
	}
	if options.CloudOrgId != "" {
		cfg.CloudContext.OrganizationId = options.CloudOrgId
		updated = true
	}

	if updated {
		return config.Save(cfg)
	}

	return nil
}

func IsUserLoggedIn(cfg config.Data, options HelmOptions) bool {
	if options.CloudUris.Api != cfg.CloudContext.ApiUri {
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

func PopulateOrgAndEnvNames(cfg config.Data, orgId, envId, rootDomain string) (config.Data, error) {
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

	orgClient := cloudclient.NewOrganizationsClient(rootDomain, cfg.CloudContext.ApiKey)
	org, err := orgClient.Get(cfg.CloudContext.OrganizationId)
	if err != nil {
		return cfg, err
	}

	envsClient := cloudclient.NewEnvironmentsClient(rootDomain, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId)
	env, err := envsClient.Get(cfg.CloudContext.EnvironmentId)
	if err != nil {
		return cfg, err
	}

	cfg.CloudContext.OrganizationName = org.Name
	cfg.CloudContext.EnvironmentName = env.Name

	return cfg, nil
}

func PopulateCloudConfig(cfg config.Data, apiKey, orgId, envId, rootDomain string) config.Data {
	if apiKey != "" {
		cfg.CloudContext.ApiKey = apiKey
	}

	// set uris based on root domain
	uris := NewCloudUris(rootDomain)
	cfg.CloudContext.ApiUri = uris.Api
	cfg.CloudContext.UiUri = uris.Ui
	cfg.CloudContext.AgentUri = uris.Agent

	var err error
	cfg, err = PopulateOrgAndEnvNames(cfg, orgId, envId, rootDomain)
	if err != nil {
		ui.Failf("Error populating org and env names: %s", err)
	}

	return cfg
}
