package export

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	chartRepoName  = "testkubeenterprise"
	chartRepoURL   = "https://kubeshop.github.io/testkube-cloud-charts"
	chartName      = "testkube-usage-export"
	defaultRelease = "testkube-usage-export"
)

type Options struct {
	Namespace       string
	KubeContext     string
	Release         string
	ValuesFiles     []string
	HelmSet         map[string]string
	HelmArg         map[string]string
	ChartVersion    string
	ChartPath       string
	Output          string
	Timeout         string
	CreateNamespace bool
	KeepRelease     bool
	DryRun          bool
}

func Install(opts Options) (string, *common.CLIError) {
	helmPath, cliErr := common.LookupHelmPath()
	if cliErr != nil {
		return "", cliErr
	}

	if opts.ChartPath == "" {
		if cliErr := updateChartRepo(helmPath, opts.DryRun); cliErr != nil {
			return "", cliErr
		}
	}

	args := []string{"upgrade", "--install", opts.Release}
	if opts.ChartPath != "" {
		args = append(args, opts.ChartPath)
	} else {
		args = append(args, fmt.Sprintf("%s/%s", chartRepoName, chartName))
	}

	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}
	if opts.CreateNamespace {
		args = append(args, "--create-namespace")
	}
	if opts.KubeContext != "" {
		args = append(args, "--kube-context", opts.KubeContext)
	}
	if opts.ChartVersion != "" {
		args = append(args, "--version", opts.ChartVersion)
	}
	for _, valuesFile := range opts.ValuesFiles {
		args = append(args, "-f", valuesFile)
	}
	for key, value := range opts.HelmSet {
		args = append(args, "--set", fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range opts.HelmArg {
		args = append(args, fmt.Sprintf("--%s", key))
		if value != "" {
			args = append(args, value)
		}
	}

	output, cliErr := common.RunHelmCommand(helmPath, args, opts.DryRun)
	if cliErr != nil {
		return "", cliErr
	}
	ui.Debug("Helm install usage-export output", output)
	return resolveJobName(opts)
}

func resolveJobName(opts Options) (string, *common.CLIError) {
	if opts.DryRun {
		return fmt.Sprintf("%s-usage-export-1", opts.Release), nil
	}

	kubectlPath, cliErr := common.LookupKubectlPath()
	if cliErr != nil {
		return "", cliErr
	}

	args := kubectlBaseArgs(opts, "get", "jobs",
		"-l", fmt.Sprintf("app.kubernetes.io/instance=%s,app.kubernetes.io/component=usage-export", opts.Release),
		"-o", "jsonpath={.items[-1].metadata.name}",
	)
	jobName, cliErr := common.RunKubectlCommand(kubectlPath, args)
	if cliErr != nil {
		return "", cliErr
	}
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return "", common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"Usage export Job not found",
			"Check helm install output and verify the release created a Job in the target namespace",
			fmt.Errorf("no job for release %q", opts.Release),
		)
	}
	return jobName, nil
}

func updateChartRepo(helmPath string, dryRun bool) *common.CLIError {
	_, err := common.RunHelmCommand(helmPath, []string{"repo", "add", chartRepoName, chartRepoURL}, dryRun)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	_, err = common.RunHelmCommand(helmPath, []string{"repo", "update"}, dryRun)
	return err
}

func kubectlBaseArgs(opts Options, args ...string) []string {
	out := make([]string, 0, len(args)+4)
	if opts.KubeContext != "" {
		out = append(out, "--context", opts.KubeContext)
	}
	if opts.Namespace != "" {
		out = append(out, "-n", opts.Namespace)
	}
	out = append(out, args...)
	return out
}
