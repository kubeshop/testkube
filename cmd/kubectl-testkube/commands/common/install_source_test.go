package common

import (
	"errors"
	"path/filepath"
	"testing"
)

// withSeams swaps the package-private detection seams (execPath, goos,
// isRunningInDocker, evalSymlinks, lookupEnv, userHomeDir) for the duration of
// the test and restores them via t.Cleanup. Pass nil for any field that should
// retain its default.
type detectStub struct {
	resolved   string
	resolveErr error
	goos       string
	inDocker   bool
	env        map[string]string
	homeDir    string
}

func withSeams(t *testing.T, stub detectStub) {
	t.Helper()

	originalExecPath := execPath
	originalGoos := goos
	originalIsRunningInDocker := isRunningInDocker
	originalEvalSymlinks := evalSymlinks
	originalLookupEnv := lookupEnv
	originalUserHomeDir := userHomeDir

	execPath = func() (string, error) {
		if stub.resolved == "" {
			return "", errors.New("no executable configured")
		}
		return stub.resolved, nil
	}
	evalSymlinks = func(path string) (string, error) {
		if stub.resolveErr != nil {
			return "", stub.resolveErr
		}
		return path, nil
	}
	goos = stub.goos
	isRunningInDocker = func() bool { return stub.inDocker }
	lookupEnv = func(key string) (string, bool) {
		if stub.env != nil {
			if v, ok := stub.env[key]; ok {
				return v, true
			}
		}
		return "", false
	}
	userHomeDir = func() (string, error) {
		if stub.homeDir == "" {
			return "", errors.New("no home dir")
		}
		return stub.homeDir, nil
	}

	t.Cleanup(func() {
		execPath = originalExecPath
		goos = originalGoos
		isRunningInDocker = originalIsRunningInDocker
		evalSymlinks = originalEvalSymlinks
		lookupEnv = originalLookupEnv
		userHomeDir = originalUserHomeDir
	})
}

func TestDetectInstallSource_Docker(t *testing.T) {
	withSeams(t, detectStub{inDocker: true, resolved: "/usr/local/bin/kubectl-testkube", goos: "linux"})
	info := DetectInstallSource()
	if info.Source != InstallSourceDocker {
		t.Fatalf("expected Docker, got %q", info.Source)
	}
	if info.UpgradeCmd == "" {
		t.Fatal("expected non-empty docker upgrade command")
	}
}

func TestDetectInstallSource_Homebrew_macOSAppleSilicon(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/opt/homebrew/Cellar/testkube/2.1.140/bin/kubectl-testkube",
		goos:     "darwin",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceHomebrew {
		t.Fatalf("expected Homebrew, got %q", info.Source)
	}
	if info.UpgradeCmd != "brew upgrade testkube" {
		t.Fatalf("unexpected upgrade command: %q", info.UpgradeCmd)
	}
}

func TestDetectInstallSource_Homebrew_Linuxbrew(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/home/linuxbrew/.linuxbrew/bin/kubectl-testkube",
		goos:     "linux",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceHomebrew {
		t.Fatalf("expected Homebrew, got %q", info.Source)
	}
}

func TestDetectInstallSource_Chocolatey(t *testing.T) {
	withSeams(t, detectStub{
		resolved: `C:\ProgramData\chocolatey\bin\kubectl-testkube.exe`,
		goos:     "windows",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceChocolatey {
		t.Fatalf("expected Chocolatey, got %q", info.Source)
	}
	if info.UpgradeCmd != "choco upgrade testkube" {
		t.Fatalf("unexpected upgrade command: %q", info.UpgradeCmd)
	}
}

func TestDetectInstallSource_Chocolatey_FromEnv(t *testing.T) {
	withSeams(t, detectStub{
		resolved: `C:\tools\choco\bin\kubectl-testkube.exe`,
		goos:     "windows",
		env:      map[string]string{"ChocolateyInstall": `C:\tools\choco`},
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceChocolatey {
		t.Fatalf("expected Chocolatey (via env), got %q", info.Source)
	}
}

func TestDetectInstallSource_Apt(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/usr/bin/kubectl-testkube",
		goos:     "linux",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceApt {
		t.Fatalf("expected Apt, got %q", info.Source)
	}
	if info.UpgradeCmd == "" {
		t.Fatal("expected non-empty apt upgrade command")
	}
}

func TestDetectInstallSource_AptOnlyOnLinux(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/usr/bin/kubectl-testkube",
		goos:     "darwin",
	})
	info := DetectInstallSource()
	if info.Source == InstallSourceApt {
		t.Fatal("apt classification must not apply on darwin")
	}
}

func TestDetectInstallSource_InstallScript(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/usr/local/bin/kubectl-testkube",
		goos:     "linux",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceInstallScript {
		t.Fatalf("expected InstallScript, got %q", info.Source)
	}
}

func TestDetectInstallSource_GoInstall_GoBinPath(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/Users/dev/go/bin/kubectl-testkube",
		goos:     "darwin",
		homeDir:  "/Users/dev",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceGoInstall {
		t.Fatalf("expected GoInstall, got %q", info.Source)
	}
}

func TestDetectInstallSource_GoInstall_FromGOPATH(t *testing.T) {
	withSeams(t, detectStub{
		resolved: filepath.Join("/srv/gopath/bin", "kubectl-testkube"),
		goos:     "linux",
		env:      map[string]string{"GOPATH": "/srv/gopath"},
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceGoInstall {
		t.Fatalf("expected GoInstall, got %q", info.Source)
	}
}

func TestDetectInstallSource_Unknown(t *testing.T) {
	withSeams(t, detectStub{
		resolved: "/opt/custom/kubectl-testkube",
		goos:     "linux",
	})
	info := DetectInstallSource()
	if info.Source != InstallSourceUnknown {
		t.Fatalf("expected Unknown, got %q", info.Source)
	}
	if info.UpgradeCmd != "" {
		t.Fatalf("expected empty upgrade command, got %q", info.UpgradeCmd)
	}
}
