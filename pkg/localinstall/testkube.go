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
	"strconv"
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
	ErrStuck           = errors.New("the Testkube install can't finish")
)

var (
	stuckPoll  = 5 * time.Second
	stuckLasts = time.Minute
)

// Waiting can't fix these; anything else may still pass.
var stopsEarly = map[string]bool{"image_missing": true, "rate_limit": true, "no_cpu": true, "no_memory": true}

type InstallState struct {
	// An earlier run was killed mid-install and got cleaned up.
	Recovered        bool
	MigrationRetried bool
	TestkubeTook     time.Duration
	RunnerTook       time.Duration
}

type Installer struct {
	helm  func(ctx context.Context, args ...string) ([]byte, error)
	apply func(ctx context.Context, manifest string) ([]byte, error)
	wait  func(ctx context.Context) ([]byte, error)
	read  func(ctx context.Context) ([]byte, error)
	logs  func(ctx context.Context, pod string, previous bool) ([]byte, error)
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
	return &Installer{helm: h.Run, apply: kubectlApply, wait: waitForPostgres, read: readCluster,
		logs: podLogs, dir: filepath.Dir(kubeconfig)}, nil
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

	// A second Ctrl+C exits before the defer below runs.
	leftovers, _ := filepath.Glob(filepath.Join(i.dir, "install-*"))
	for _, dir := range leftovers {
		_ = os.RemoveAll(dir)
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
	began := start
	install := func() ([]byte, *Stuck, error) {
		return i.watchedHelm(ctx, began, "upgrade", "--install", ReleaseName, EnterpriseChart, "--version", EnterpriseChartVersion,
			"--namespace", Namespace, "--create-namespace", "-f", values["demo"], "-f", values["ports"], "-f", values["demo-fixes"], "-f", values["private"],
			"--wait", "--wait-for-jobs", "--timeout", "15m")
	}
	out, stuck, err := install()
	// The chart's migration retries ~1 minute; postgres can take longer.
	if err != nil && stuck == nil && migrationLostRace(string(out)) {
		state.MigrationRetried = true
		// Best effort: a postgres that never starts fails the retry.
		_, _ = i.wait(ctx)
		out, stuck, err = install()
	}
	if err != nil && stuck != nil {
		return state, string(out), StuckError{Stuck: *stuck, After: time.Since(began), err: fmt.Errorf("%w: %w", ErrTestkubeInstall, ErrStuck)}
	}
	if err != nil {
		return state, string(out), i.explain(ctx, string(out), classify(string(out), ErrTestkubeInstall), began)
	}
	state.TestkubeTook, start = time.Since(start), time.Now()
	out, stuck, err = i.watchedHelm(ctx, began, "upgrade", "--install", runnerRelease, RunnerChart, "--version", RunnerChartVersion,
		"--namespace", Namespace, "-f", values["runner"], "--wait", "--timeout", "10m")
	if err != nil && stuck != nil {
		return state, string(out), StuckError{Stuck: *stuck, After: time.Since(began), err: fmt.Errorf("%w: %w", ErrRunnerInstall, ErrStuck)}
	}
	if err != nil {
		return state, string(out), i.explain(ctx, string(out), classify(string(out), ErrRunnerInstall), began)
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

// Slow downloads can keep postgres starting for minutes.
func waitForPostgres(ctx context.Context) ([]byte, error) {
	return runCombined(ctx, "docker", "exec", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"--namespace", Namespace, "wait", "--for=condition=Ready", "pod/testkube-enterprise-postgresql-0", "--timeout=10m")
}

// Carried on the install error, so errors.Is checks still match.
type StuckError struct {
	Stuck
	// Set when we stopped helm early.
	After time.Duration
	err   error
}

// Cancels only helm's own context, so it never looks like Ctrl+C.
func (i *Installer) watchedHelm(ctx context.Context, since time.Time, args ...string) ([]byte, *Stuck, error) {
	helmCtx, stopHelm := context.WithCancel(ctx)
	defer stopHelm()
	found := make(chan Stuck, 1)
	done := make(chan struct{})
	poll, lasts := stuckPoll, stuckLasts
	go func() {
		ticker := time.NewTicker(poll)
		defer ticker.Stop()
		var seen Stuck
		var seenAt time.Time
		for {
			select {
			case <-done:
				return
			case <-helmCtx.Done():
				return
			case <-ticker.C:
			}
			// Logs only matter for the final message.
			st, ok := i.diagnose(since, false)
			switch {
			case !ok || !stopsEarly[st.Reason]:
				seen = Stuck{}
			case st.Service != seen.Service || st.Reason != seen.Reason:
				seen, seenAt = st, time.Now()
			// A pod still terminating can block scheduling briefly.
			case time.Since(seenAt) >= lasts:
				found <- st
				stopHelm()
				return
			}
		}
	}()
	out, err := i.helm(helmCtx, args...)
	close(done)
	if err == nil {
		return out, nil, nil
	}
	select {
	case st := <-found:
		return out, &st, err
	default:
		return out, nil, err
	}
}

func (e StuckError) Error() string { return e.err.Error() }
func (e StuckError) Unwrap() error { return e.err }

// Only waits involve pods; after Ctrl+C nobody reads the answer.
func (i *Installer) explain(ctx context.Context, out string, err error, since time.Time) error {
	if ctx.Err() != nil || !errors.Is(err, ErrInstallTimeout) && !strings.Contains(out, " not ready. status: ") {
		return err
	}
	if st, ok := i.diagnose(since, true); ok {
		return StuckError{Stuck: st, err: err}
	}
	return err
}

// Fresh context: the install's own may be done by now.
func (i *Installer) diagnose(since time.Time, withLogs bool) (Stuck, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	data, err := i.read(ctx)
	if err != nil {
		return Stuck{}, false
	}
	s, err := parseSnapshot(data)
	if err != nil {
		return Stuck{}, false
	}
	st, ok := findStuck(s, since, time.Now())
	if ok && withLogs && st.Pod != "" && (st.Reason == "crashloop" || st.Reason == "job_failed") {
		st.Logs = i.lastLogLines(ctx, st.Pod)
	}
	return st, ok
}

// The current container is often the dead one; previous may lie.
func (i *Installer) lastLogLines(ctx context.Context, pod string) []string {
	for _, previous := range []bool{false, true} {
		out, err := i.logs(ctx, pod, previous)
		if err != nil || strings.Contains(string(out), "unable to retrieve container logs") {
			continue
		}
		if lines := readableLogLines(string(out)); len(lines) > 0 {
			return lines
		}
	}
	return nil
}

// Stdout only: kubectl's notes on stderr aren't the pod's logs.
func podLogs(ctx context.Context, pod string, previous bool) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", "exec", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"--namespace", Namespace, "logs", pod, "--tail=3", "--previous="+strconv.FormatBool(previous)).Output()
}

// Stdout only: kubectl warnings on stderr would break the JSON.
func readCluster(ctx context.Context) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", "exec", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"--namespace", Namespace, "get", "pods,jobs,events", "-o", "json").Output()
}

func migrationLostRace(out string) bool {
	return strings.Contains(out, "api-migration") && strings.Contains(out, "status: Failed")
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
	fixes, err := demoFixValues()
	if err != nil {
		return nil, err
	}
	paths := map[string]string{}
	for name, data := range map[string][]byte{"demo": EnterpriseDemoValues, "ports": []byte(PortValues(ports)),
		"demo-fixes": fixes, "private": private, "runner": runner} {
		paths[name] = filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(paths[name], data, 0o600); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// Upstream demo values miss these; drop each once fixed there.
func demoFixValues() ([]byte, error) {
	var demo map[string]any
	if err := yaml.Unmarshal(EnterpriseDemoValues, &demo); err != nil {
		return nil, err
	}
	api, _ := dig(demo, "testkube-cloud-api", "api").(map[string]any)
	if api["mongo"] == nil || api["postgres"] == nil {
		return nil, errors.New("demo values have no API database")
	}
	return yaml.Marshal(map[string]any{
		// Otherwise it waits for Mongo, which isn't installed.
		"testkube-worker-service": map[string]any{"api": map[string]any{"mongo": api["mongo"], "postgres": api["postgres"]}},
		// Otherwise built from a domain we don't have.
		"global": map[string]any{"ai": map[string]any{"serviceUrl": "http://testkube-enterprise-ai-service:9090"}},
	})
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
