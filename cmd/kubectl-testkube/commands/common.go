package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RunMigrations(cmd *cobra.Command) (hasMigrations bool, err error) {
	client, _ := common.GetClient(cmd)
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

type HelmUpgradeOrInstalTestkubeOptions struct {
	Name, Namespace, Chart, Values, AgentToken, AgentUri string
	NoDashboard, NoMinio, NoMongo, NoConfirm             bool
	MinioReplicas, MongoReplicas, DashboardReplicas      int
	DryRun                                               bool
	CloudAgentToken                                      string
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

func HelmUpgradeOrInstallTestkubeCloud(options HelmUpgradeOrInstalTestkubeOptions, cfg config.Data) error {
	// use config if set
	if cfg.CloudContext.AgentKey != "" && options.AgentToken == "" {
		options.AgentToken = cfg.CloudContext.AgentKey
	}
	if cfg.CloudContext.AgentUri != "" && options.AgentUri == "" {
		options.AgentUri = cfg.CloudContext.AgentUri
	}

	if options.AgentToken == "" || options.AgentUri == "" {
		return fmt.Errorf("agentKey and agentUri are required, please pass it with `--agent-token` and `--agent-uri` flags")
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
		"--set", "testkube-api.cloud.url=" + options.AgentUri,
		"--set", "testkube-api.cloud.key=" + options.AgentToken,
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

func HelmUpgradeOrInstalTestkube(options HelmUpgradeOrInstalTestkubeOptions) error {
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

func PopulateUpgradeInstallFlags(cmd *cobra.Command, options *HelmUpgradeOrInstalTestkubeOptions) {
	cmd.Flags().StringVar(&options.Chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&options.Name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().StringVar(&options.Values, "values", "", "path to Helm values file")

	cmd.Flags().StringVar(&options.AgentToken, "agent-token", "", "Testkube Cloud agent key [required for cloud mode]")
	cmd.Flags().StringVar(&options.AgentUri, "agent-uri", "agent.testkube.io:443", "Testkube Cloud agent URI [required for cloud mode]")

	cmd.Flags().BoolVar(&options.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&options.NoDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&options.NoMongo, "no-mongo", false, "don't install MongoDB")
	cmd.Flags().BoolVar(&options.NoConfirm, "no-confirm", false, "don't ask for confirmation - unatended installation mode")

	cmd.Flags().IntVar(&options.MinioReplicas, "minio-replicas", 1, "Scale MinIO replicas")
	cmd.Flags().IntVar(&options.MongoReplicas, "mongo-replicas", 1, "Scale MongoDB replicas")
	cmd.Flags().IntVar(&options.DashboardReplicas, "dashboard-replicas", 1, "Don't install MongoDB")

	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "dry run mode - only print commands that would be executed")
}

func PopulateAgentDataToContext(options HelmUpgradeOrInstalTestkubeOptions, cfg config.Data) error {
	updated := false
	if options.AgentToken != "" {
		cfg.CloudContext.AgentKey = options.AgentToken
		updated = true
	}
	if options.AgentUri != "" {
		cfg.CloudContext.AgentUri = options.AgentUri
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
