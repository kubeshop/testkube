package localinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const runnerRelease = "testkube-demo-runner"

var (
	ErrTestkubePrepare = errors.New("the cluster could not be prepared for Testkube")
	ErrTestkubeInstall = errors.New("the Testkube install failed")
	ErrRunnerInstall   = errors.New("the runner could not be installed")
	ErrChartDownload   = errors.New("the chart could not be downloaded")
	ErrInstallTimeout  = errors.New("the Testkube install was not ready in time")
)

type InstallState struct {
	// An earlier run was killed mid-install and got cleaned up.
	Recovered    bool
	TestkubeTook time.Duration
	RunnerTook   time.Duration
}

type Installer struct {
	helm  func(ctx context.Context, args ...string) ([]byte, error)
	apply func(ctx context.Context, manifest string) ([]byte, error)
	dir   string
}

func NewInstaller() (*Installer, error) {
	h, err := NewHelm()
	if err != nil {
		return nil, err
	}
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return nil, err
	}
	return &Installer{helm: h.Run, apply: kubectlApply, dir: filepath.Dir(kubeconfig)}, nil
}

func (i *Installer) Install(ctx context.Context, ports Ports, s Secrets, license string) (InstallState, string, error) {
	var state InstallState
	if out, err := i.apply(ctx, preinstallManifest(s)); err != nil {
		return state, string(out), ErrTestkubePrepare
	}
	for _, release := range []string{ReleaseName, runnerRelease} {
		recovered, out, err := i.recover(ctx, release)
		if err != nil {
			return state, out, ErrTestkubePrepare
		}
		state.Recovered = state.Recovered || recovered
	}

	files, err := os.MkdirTemp(i.dir, "install-")
	if err != nil {
		return state, "", err
	}
	// The license and keys must not outlive the install.
	defer os.RemoveAll(files)
	values, err := writeValues(files, ports, s, license)
	if err != nil {
		return state, "", err
	}

	start := time.Now()
	out, err := i.helm(ctx, "upgrade", "--install", ReleaseName, EnterpriseChart, "--version", EnterpriseChartVersion,
		"--namespace", Namespace, "--create-namespace", "-f", values["demo"], "-f", values["ports"], "-f", values["private"],
		"--wait", "--wait-for-jobs", "--timeout", "15m")
	if err != nil {
		return state, string(out), classify(string(out), ErrTestkubeInstall)
	}
	state.TestkubeTook, start = time.Since(start), time.Now()
	out, err = i.helm(ctx, "upgrade", "--install", runnerRelease, RunnerChart, "--version", RunnerChartVersion,
		"--namespace", Namespace, "-f", values["runner"], "--wait", "--timeout", "10m")
	if err != nil {
		return state, string(out), classify(string(out), ErrRunnerInstall)
	}
	state.RunnerTook = time.Since(start)
	return state, "", nil
}

// A killed helm leaves a pending release that blocks upgrades.
func (i *Installer) recover(ctx context.Context, release string) (bool, string, error) {
	out, err := i.helm(ctx, "list", "--namespace", Namespace, "--pending", "--uninstalling", "--filter", "^"+release+"$", "-o", "json")
	if err != nil {
		return false, string(out), err
	}
	var releases []struct{ Status string }
	if err := json.Unmarshal(out, &releases); err != nil {
		return false, string(out), err
	}
	if len(releases) == 0 {
		return false, "", nil
	}
	if status := releases[0].Status; !strings.HasPrefix(status, "pending-") && status != "uninstalling" {
		return false, "", nil
	}
	// Data survives: volumes are kept and Secrets aren't helm's.
	out, err = i.helm(ctx, "uninstall", release, "--namespace", Namespace, "--wait", "--timeout", "5m")
	return err == nil, string(out), err
}

func classify(out string, stage error) error {
	switch {
	case strings.Contains(out, "context deadline exceeded") || strings.Contains(out, "timed out waiting"):
		return fmt.Errorf("%w: %w", stage, ErrInstallTimeout)
	case strings.Contains(out, "failed to do request") || strings.Contains(out, "failed to fetch") ||
		strings.Contains(out, "no such host"):
		return fmt.Errorf("%w: %w", stage, ErrChartDownload)
	}
	return stage
}

// The chart's generator keeps these when they already exist.
func preinstallManifest(s Secrets) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %[1]s
---
apiVersion: v1
kind: Secret
metadata:
  name: testkube-credentials-master
  namespace: %[1]s
stringData:
  password: %[2]q
---
apiVersion: v1
kind: Secret
metadata:
  name: testkube-minio-credentials
  namespace: %[1]s
stringData:
  root-user: testkube-enterprise
  root-password: %[3]q
  token: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: testkube-ai-internal-secret
  namespace: %[1]s
stringData:
  secret: %[4]q
---
`, Namespace, s.MasterPassword, s.MinioPassword, s.AIToken) + ServicesManifest()
}

func writeValues(dir string, ports Ports, s Secrets, license string) (map[string]string, error) {
	private, err := enterprisePrivateValues(s.RunnerKey, license)
	if err != nil {
		return nil, err
	}
	runner, err := yaml.Marshal(map[string]any{
		"fullnameOverride": runnerRelease,
		"runner": map[string]any{"id": "tkcrun_demo-runner", "name": "demo-runner", "orgId": "tkcorg_demo",
			"envId": "tkcenv_my-first-environment", "secret": s.RunnerKey},
		"cloud":    map[string]any{"url": "testkube-enterprise-api." + Namespace + ".svc.cluster.local:8089", "tls": map[string]any{"enabled": false}},
		"listener": map[string]any{"enabled": true},
	})
	if err != nil {
		return nil, err
	}
	paths := map[string]string{}
	for name, data := range map[string][]byte{"demo": EnterpriseDemoValues, "ports": []byte(PortValues(ports)),
		"private": private, "runner": runner} {
		paths[name] = filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(paths[name], data, 0o600); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// Helm replaces lists whole, so the runner list is copied.
func enterprisePrivateValues(runnerKey, license string) ([]byte, error) {
	var demo map[string]any
	if err := yaml.Unmarshal(EnterpriseDemoValues, &demo); err != nil {
		return nil, err
	}
	bootstrap, _ := dig(demo, "testkube-cloud-api", "api", "features", "bootstrapConfig").(map[string]any)
	orgs, _ := dig(bootstrap, "config", "organizations").([]any)
	if len(orgs) == 0 {
		return nil, errors.New("demo values have no bootstrap organization")
	}
	runners, _ := dig(orgs[0], "runners").([]any)
	if len(runners) == 0 {
		return nil, errors.New("demo values have no bootstrap runner")
	}
	runner, ok := runners[0].(map[string]any)
	if !ok {
		return nil, errors.New("demo values have no bootstrap runner")
	}
	runner["secret_key"] = runnerKey
	return yaml.Marshal(map[string]any{
		"global":             map[string]any{"enterpriseLicenseKey": license},
		"testkube-cloud-api": map[string]any{"api": map[string]any{"features": map[string]any{"bootstrapConfig": bootstrap}}},
	})
}

func dig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// Through stdin: Secrets in arguments would show up in ps.
func kubectlApply(ctx context.Context, manifest string) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.Bytes(), err
}
