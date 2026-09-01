package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	defaultImageRegistry   = "docker.io"
	defaultImageRepository = "kubeshop/testkube-usage-export"
	defaultImageTag        = "2.12.0-rc.0"
)

type runVersions struct {
	CLI   string
	Chart string
	Image string
}

type chartFileMeta struct {
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

func resolveRunVersions(opts Options) runVersions {
	versions := runVersions{
		CLI:   strings.TrimSpace(common.Version),
		Chart: resolveChartVersion(opts),
		Image: resolveImageReference(opts, chartAppVersion(opts)),
	}
	if versions.CLI == "" {
		versions.CLI = "unknown"
	}
	return versions
}

func chartAppVersion(opts Options) string {
	if meta, ok := readChartMeta(opts.ChartPath); ok && meta.AppVersion != "" {
		return meta.AppVersion
	}
	return defaultImageTag
}

func resolveChartVersion(opts Options) string {
	if opts.ChartVersion != "" {
		return opts.ChartVersion
	}
	if meta, ok := readChartMeta(opts.ChartPath); ok {
		if meta.Version != "" {
			if opts.ChartPath != "" {
				return fmt.Sprintf("%s (%s)", meta.Version, filepath.Base(opts.ChartPath))
			}
			return meta.Version
		}
	}
	if opts.ChartPath != "" {
		return fmt.Sprintf("local (%s)", filepath.Base(opts.ChartPath))
	}
	return fmt.Sprintf("%s/%s (latest)", chartRepoName, chartName)
}

func readChartMeta(chartPath string) (chartFileMeta, bool) {
	if chartPath == "" {
		return chartFileMeta{}, false
	}
	raw, err := os.ReadFile(filepath.Join(chartPath, "Chart.yaml"))
	if err != nil {
		return chartFileMeta{}, false
	}
	var meta chartFileMeta
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return chartFileMeta{}, false
	}
	return meta, true
}

func resolveImageReference(opts Options, defaultTag string) string {
	registry := helmSetValue(opts.HelmSet, "image.registry", defaultImageRegistry)
	repository := helmSetValue(opts.HelmSet, "image.repository", defaultImageRepository)
	tag := helmSetValue(opts.HelmSet, "image.tag", defaultTag)
	tagSuffix := helmSetValue(opts.HelmSet, "image.tagSuffix", "")
	digest := helmSetValue(opts.HelmSet, "image.digest", "")

	if digest != "" {
		return fmt.Sprintf("%s/%s@%s", registry, repository, digest)
	}
	return fmt.Sprintf("%s/%s:%s%s", registry, repository, tag, tagSuffix)
}

func helmSetValue(set map[string]string, key, fallback string) string {
	if set == nil {
		return fallback
	}
	if v, ok := set[key]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func printRunVersions(opts Options) {
	v := resolveRunVersions(opts)
	ui.NL()
	ui.H2("Versions")
	ui.Info("CLI:", v.CLI)
	ui.Info("Chart:", v.Chart)
	ui.Info("Image:", v.Image)
	ui.NL()
}

func finishUsageExport(opts Options, jobName, podName string, cliErr *common.CLIError) {
	printRunVersions(opts)
	printExportLogs(opts, jobName, podName)
	common.HandleCLIError(cliErr)
}

func printExportLogs(opts Options, jobName, podName string) {
	if opts.DryRun {
		return
	}
	logs, source := exportLogs(opts, jobName, podName)
	if logs == "" {
		return
	}
	ui.H2("Export logs")
	if source != "" {
		ui.Info(source)
	}
	ui.Info(logs)
	ui.NL()
}

func exportLogs(opts Options, jobName, podName string) (logs, source string) {
	kubectlPath, cliErr := common.LookupKubectlPath()
	if cliErr != nil {
		return "", ""
	}
	if podName == "" && jobName != "" {
		podName = exportPodNameForJob(kubectlPath, opts, jobName)
	}
	if podName == "" {
		return "", ""
	}
	logs, cliErr = podLogs(kubectlPath, opts, podName)
	if cliErr != nil || strings.TrimSpace(logs) == "" {
		return "", ""
	}
	source = fmt.Sprintf("From pod %s:", podName)
	return strings.TrimSpace(logs), source
}

func exportPodNameForJob(kubectlPath string, opts Options, jobName string) string {
	args := kubectlBaseArgs(opts, "get", "pods",
		"-l", fmt.Sprintf("job-name=%s", jobName),
		"-o", "jsonpath={.items[0].metadata.name}",
	)
	podName, cliErr := common.RunKubectlCommand(kubectlPath, args)
	if cliErr != nil {
		return ""
	}
	return strings.TrimSpace(podName)
}
