package common

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kubeshop/testkube/pkg/cliruntime"
)

// InstallSource is a discrete classification of how the running CLI binary was
// installed. It is used to surface the appropriate package-manager or
// installer-specific upgrade command in the update-check hint.
type InstallSource string

const (
	InstallSourceHomebrew      InstallSource = "homebrew"
	InstallSourceChocolatey    InstallSource = "chocolatey"
	InstallSourceApt           InstallSource = "apt"
	InstallSourceInstallScript InstallSource = "install-script"
	InstallSourceDocker        InstallSource = "docker"
	InstallSourceGoInstall     InstallSource = "go-install"
	InstallSourceUnknown       InstallSource = "unknown"
)

// InstallInfo describes the detected origin of the running CLI binary.
type InstallInfo struct {
	Source InstallSource
	// UpgradeCmd is the command a user can run to upgrade this CLI. Empty for
	// InstallSourceUnknown; callers should fall back to a "download from
	// releases page" message in that case.
	UpgradeCmd string
	// ResolvedPath is the filesystem path the executable resolves to after
	// following symlinks. Surfaced only via verbose/debug output for
	// diagnostics; empty when the path could not be resolved.
	ResolvedPath string
}

// These package-private seams keep DetectInstallSource fully unit-testable
// without requiring real binaries on disk or platform-specific runners.
var (
	execPath            = os.Executable
	goos                = runtime.GOOS
	isRunningInDocker   = cliruntime.IsRunningInDocker
	evalSymlinks        = filepath.EvalSymlinks
	lookupEnv           = os.LookupEnv
	userHomeDir         = os.UserHomeDir
	releasesPageURL     = "https://github.com/kubeshop/testkube/releases/latest"
	homebrewUpgradeCmd  = "brew upgrade testkube"
	chocoUpgradeCmd     = "choco upgrade testkube"
	aptUpgradeCmd       = "sudo apt-get update && sudo apt-get install --only-upgrade testkube"
	scriptUpgradeCmd    = "curl -sSLf https://get.testkube.io | sh"
	dockerUpgradeCmd    = "docker pull kubeshop/testkube-cli:latest"
	goInstallUpgradeCmd = "go install github.com/kubeshop/testkube/cmd/kubectl-testkube@latest"
)

// DetectInstallSource attempts to classify how the running CLI binary was
// installed. The classification is best-effort and falls back to
// InstallSourceUnknown when no heuristic matches. The returned UpgradeCmd is
// always safe to display verbatim to the user.
func DetectInstallSource() InstallInfo {
	if isRunningInDocker() {
		return InstallInfo{Source: InstallSourceDocker, UpgradeCmd: dockerUpgradeCmd}
	}

	exe, err := execPath()
	if err != nil || exe == "" {
		return InstallInfo{Source: InstallSourceUnknown}
	}

	resolved, err := evalSymlinks(exe)
	if err != nil || resolved == "" {
		resolved = exe
	}

	switch goos {
	case "windows":
		if matched := classifyWindows(resolved); matched.Source != "" {
			matched.ResolvedPath = resolved
			return matched
		}
	default:
		if matched := classifyUnix(resolved); matched.Source != "" {
			matched.ResolvedPath = resolved
			return matched
		}
	}

	if matched := classifyGoInstall(resolved); matched.Source != "" {
		matched.ResolvedPath = resolved
		return matched
	}

	return InstallInfo{Source: InstallSourceUnknown, ResolvedPath: resolved}
}

func classifyWindows(resolved string) InstallInfo {
	lower := strings.ToLower(resolved)
	if strings.Contains(lower, `\chocolatey\`) || strings.Contains(lower, `/chocolatey/`) {
		return InstallInfo{Source: InstallSourceChocolatey, UpgradeCmd: chocoUpgradeCmd}
	}
	if chocoRoot, ok := lookupEnv("ChocolateyInstall"); ok && chocoRoot != "" {
		if strings.HasPrefix(lower, strings.ToLower(chocoRoot)) {
			return InstallInfo{Source: InstallSourceChocolatey, UpgradeCmd: chocoUpgradeCmd}
		}
	}
	return InstallInfo{}
}

func classifyUnix(resolved string) InstallInfo {
	// Homebrew binaries always resolve under a Cellar directory regardless of
	// the user-facing prefix, so the Cellar check covers Intel Macs,
	// Apple-silicon Macs, and Linuxbrew installs equally well.
	if strings.Contains(resolved, "/Cellar/") {
		return InstallInfo{Source: InstallSourceHomebrew, UpgradeCmd: homebrewUpgradeCmd}
	}
	homebrewPrefixes := []string{
		"/opt/homebrew/",
		"/usr/local/Homebrew/",
		"/home/linuxbrew/.linuxbrew/",
	}
	if prefix, ok := lookupEnv("HOMEBREW_PREFIX"); ok && prefix != "" {
		homebrewPrefixes = append(homebrewPrefixes, strings.TrimRight(prefix, "/")+"/")
	}
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(resolved, prefix) {
			return InstallInfo{Source: InstallSourceHomebrew, UpgradeCmd: homebrewUpgradeCmd}
		}
	}

	// The install.sh script always drops the binary in /usr/local/bin (see
	// install.sh:119) while the deb package puts it in /usr/bin. We classify
	// in that order because /usr/local/bin matches both Linux and macOS, but
	// /usr/bin only makes sense as "installed via apt" on Linux.
	if resolved == "/usr/local/bin/kubectl-testkube" {
		return InstallInfo{Source: InstallSourceInstallScript, UpgradeCmd: scriptUpgradeCmd}
	}
	if goos == "linux" && strings.HasPrefix(resolved, "/usr/bin/") {
		return InstallInfo{Source: InstallSourceApt, UpgradeCmd: aptUpgradeCmd}
	}

	return InstallInfo{}
}

func classifyGoInstall(resolved string) InstallInfo {
	if strings.Contains(resolved, "/go/bin/") || strings.Contains(resolved, `\go\bin\`) {
		return InstallInfo{Source: InstallSourceGoInstall, UpgradeCmd: goInstallUpgradeCmd}
	}
	if gopath, ok := lookupEnv("GOPATH"); ok && gopath != "" {
		binDir := filepath.Join(gopath, "bin")
		if strings.HasPrefix(resolved, binDir+string(filepath.Separator)) {
			return InstallInfo{Source: InstallSourceGoInstall, UpgradeCmd: goInstallUpgradeCmd}
		}
	}
	if home, err := userHomeDir(); err == nil && home != "" {
		binDir := filepath.Join(home, "go", "bin")
		if strings.HasPrefix(resolved, binDir+string(filepath.Separator)) {
			return InstallInfo{Source: InstallSourceGoInstall, UpgradeCmd: goInstallUpgradeCmd}
		}
	}
	return InstallInfo{}
}

// ReleasesPageURL returns the canonical "latest releases" URL used as the
// fallback hint when the install source cannot be determined.
func ReleasesPageURL() string {
	return releasesPageURL
}
