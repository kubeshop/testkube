package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
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
	Name, Namespace, Chart, Values string
	NoDashboard, NoMinio, NoMongo  bool
}

func HelmUpgradeOrInstalTestkube(options HelmUpgradeOrInstalTestkubeOptions) error {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return err
	}

	ui.Info("Helm installing testkube framework")
	_, err = process.Execute(helmPath, "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
		ui.WarnOnError("adding testkube repo", err)
	}

	_, err = process.Execute(helmPath, "repo", "update")
	ui.ExitOnError("updating helm repositories", err)

	command := []string{"upgrade", "--install", "--create-namespace", "--namespace", options.Namespace}
	command = append(command, "--set", fmt.Sprintf("testkube-api.minio.enabled=%t", !options.NoMinio))
	if options.NoMinio {
		command = append(command, "--set", "testkube-api.logs.storage=mongo")
	} else {
		command = append(command, "--set", "testkube-api.logs.storage=minio")
	}
	command = append(command, "--set", fmt.Sprintf("testkube-dashboard.enabled=%t", !options.NoDashboard))
	command = append(command, "--set", fmt.Sprintf("mongodb.enabled=%t", !options.NoMongo))
	command = append(command, options.Name, options.Chart)

	if options.Values != "" {
		command = append(command, "--values", options.Values)
	}

	out, err := process.Execute(helmPath, command...)
	if err != nil {
		return err
	}

	ui.Info("Helm install testkube output", string(out))
	return nil
}

func PopulateUpgradeInstallFlags(cmd *cobra.Command, options *HelmUpgradeOrInstalTestkubeOptions) {
	cmd.Flags().StringVar(&options.Chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&options.Name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().StringVar(&options.Values, "values", "", "path to Helm values file")

	cmd.Flags().BoolVar(&options.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&options.NoDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&options.NoMongo, "no-mongo", false, "don't install MongoDB")
}
