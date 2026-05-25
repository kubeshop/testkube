package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cliruntime"
	"github.com/kubeshop/testkube/pkg/ui"
)

// outputPrettyValue mirrors render.OutputPretty without importing the render
// package (which imports common, so a direct dependency would create an import
// cycle). Kept in sync with cmd/kubectl-testkube/commands/common/render.
const outputPrettyValue = "pretty"

const (
	// updateCheckInterval is the minimum time between successive GitHub release
	// lookups for the per-command hint path. The user-explicit `testkube
	// version` path ignores this cache.
	updateCheckInterval = 24 * time.Hour
	// updateCheckTimeout caps every GitHub request issued by the update-check
	// feature so a slow or unreachable network never delays the CLI.
	updateCheckTimeout = 1500 * time.Millisecond
	// updateCheckEnvDisable, when set to any non-empty value, suppresses both
	// the per-command hint and the version-command status block.
	updateCheckEnvDisable = "TESTKUBE_DISABLE_UPDATE_CHECK"
)

// fetchLatestVersion is the indirection used by MaybeNotifyNewerRelease and
// CheckComponentsStatus so unit tests can substitute a stub fetcher.
var fetchLatestVersion = func(ctx context.Context) (string, error) {
	return GetLatestVersionWithContext(ctx)
}

// detectCliRunContext is the indirection used by MaybeNotifyNewerRelease so
// unit tests can simulate a "local" execution context regardless of the host
// environment. Production callers always hit cliruntime.CliRunContext.
var detectCliRunContext = cliruntime.CliRunContext

// MaybeNotifyNewerRelease optionally prints a one-line hint when a newer
// Testkube CLI release is available. It is designed to be called from
// PersistentPostRun on every command and short-circuits in any automated
// context (CI, Docker, Kubernetes pods, etc.) and for the `version` command
// (which renders its own richer report). The returned bool indicates whether
// cfg was mutated and should be persisted by the caller.
func MaybeNotifyNewerRelease(cmd *cobra.Command, cfg *config.Data) bool {
	if cmd != nil && cmd.Name() == "version" {
		return false
	}
	if !updateCheckEnabled(cmd) {
		return false
	}
	if detectCliRunContext() != cliruntime.CliRunContextLocal {
		return false
	}

	currentVersion, ok := parseSemverLoose(Version)
	if !ok {
		return false
	}

	latest, dirty := latestVersionForHint(cfg)
	if latest == "" {
		return false
	}

	latestVersion, ok := parseSemverLoose(latest)
	if !ok {
		return false
	}

	if !currentVersion.LessThan(latestVersion) {
		return dirty
	}

	printCliUpgradeHint(latest, Version)
	return dirty
}

// CheckComponentsStatus prints a per-component update status block for the
// `testkube version` command. Unlike MaybeNotifyNewerRelease it always runs an
// online lookup (subject to the env opt-out and output-format gates) because
// the user explicitly asked for version information. It returns true when cfg
// was mutated so the caller can persist the refreshed cache. ContextType drives
// whether the server block is rendered.
func CheckComponentsStatus(cmd *cobra.Command, cfg *config.Data, cliVersion, serverVersion string) bool {
	if !updateCheckEnabled(cmd) {
		return false
	}

	ui.NL()
	ui.H1("Update Status")

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	dirty := false
	if err != nil {
		ui.Debug("update check: failed to fetch latest release", err.Error())
	} else if latest != "" && cfg != nil {
		cfg.LatestKnownVersion = latest
		cfg.LastUpdateCheckAt = time.Now()
		dirty = true
	}

	printCliStatus(cliVersion, latest)
	printServerStatus(cfg, serverVersion, latest)
	return dirty
}

// printCliUpgradeHint emits the install-source-aware "new release available"
// message used by both MaybeNotifyNewerRelease and CheckComponentsStatus.
func printCliUpgradeHint(latest, current string) {
	info := DetectInstallSource()
	ui.NL()
	ui.Info("A new Testkube CLI release is available", latest,
		fmt.Sprintf("(you are running %s)", current))
	if info.UpgradeCmd != "" {
		ui.Hint(fmt.Sprintf("Upgrade with: %s", info.UpgradeCmd))
	} else {
		ui.Hint(fmt.Sprintf("Download the latest binary from %s", ReleasesPageURL()))
	}
	if ui.IsVerbose() && info.ResolvedPath != "" {
		ui.Debug("Detected install source", string(info.Source), info.ResolvedPath)
	}
}

func printCliStatus(cliVersion, latest string) {
	current, ok := parseSemverLoose(cliVersion)
	if !ok {
		// Dev/local builds: report the raw version we have without comparing.
		ui.Info("Testkube CLI", fallback(cliVersion, "unknown"))
		return
	}
	if latest == "" {
		ui.Info("Testkube CLI", cliVersion, "(latest: unknown)")
		return
	}
	latestVersion, ok := parseSemverLoose(latest)
	if !ok {
		ui.Info("Testkube CLI", cliVersion, "(latest: unknown)")
		return
	}
	if current.LessThan(latestVersion) {
		printCliUpgradeHint(latest, cliVersion)
		return
	}
	ui.Info("Testkube CLI", cliVersion, "(up to date)")
}

func printServerStatus(cfg *config.Data, serverVersion, latest string) {
	if cfg == nil || serverVersion == "" {
		if cfg != nil && cfg.ContextType == config.ContextTypeKubeconfig {
			ui.Debug("update check: Testkube server is not reachable, skipping server status")
		}
		return
	}

	if cfg.ContextType == config.ContextTypeCloud {
		ui.Info("Testkube Server", serverVersion, "(managed by Testkube Cloud)")
		return
	}

	if cfg.ContextType != config.ContextTypeKubeconfig {
		return
	}

	current, ok := parseSemverLoose(serverVersion)
	if !ok {
		ui.Info("Testkube Server", serverVersion)
		return
	}
	if latest == "" {
		ui.Info("Testkube Server", serverVersion, "(latest: unknown)")
		return
	}
	latestVersion, ok := parseSemverLoose(latest)
	if !ok {
		ui.Info("Testkube Server", serverVersion, "(latest: unknown)")
		return
	}
	if current.LessThan(latestVersion) {
		ui.NL()
		ui.Info("Testkube Server", serverVersion, fmt.Sprintf("(latest: %s)", latest))
		ui.Hint("Upgrade the cluster install with: testkube upgrade")
		return
	}
	ui.Info("Testkube Server", serverVersion, "(up to date)")
}

// updateCheckEnabled checks the universal preconditions shared by both
// MaybeNotifyNewerRelease and CheckComponentsStatus: pretty output is required
// (so we never break -o json/yaml) and the env opt-out must not be set.
func updateCheckEnabled(cmd *cobra.Command) bool {
	if v, _ := os.LookupEnv(updateCheckEnvDisable); v != "" {
		return false
	}
	return outputIsPretty(cmd)
}

func outputIsPretty(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	outputFlag := cmd.Flag("output")
	if outputFlag == nil {
		return true
	}
	value := outputFlag.Value.String()
	if value == "" {
		return true
	}
	return value == outputPrettyValue
}

// latestVersionForHint returns the latest known release version for the
// per-command hint path, honouring the 24h cache. It only performs a network
// call when the cache is stale or empty. The second return value indicates
// whether cfg was refreshed with new data.
func latestVersionForHint(cfg *config.Data) (string, bool) {
	if cfg != nil && cfg.LatestKnownVersion != "" &&
		time.Since(cfg.LastUpdateCheckAt) < updateCheckInterval {
		return cfg.LatestKnownVersion, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	if err != nil || latest == "" {
		if err != nil {
			ui.Debug("update check: failed to fetch latest release", err.Error())
		}
		return "", false
	}
	if cfg != nil {
		cfg.LatestKnownVersion = latest
		cfg.LastUpdateCheckAt = time.Now()
	}
	return latest, cfg != nil
}

// parseSemverLoose accepts both bare ("1.2.3") and "v"-prefixed inputs and
// returns the parsed value plus an "ok" flag. Empty input returns ok=false so
// callers can use it as a single combined "is comparable" guard.
func parseSemverLoose(input string) (*semver.Version, bool) {
	if input == "" {
		return nil, false
	}
	v, err := semver.NewVersion(input)
	if err != nil {
		return nil, false
	}
	return v, true
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
