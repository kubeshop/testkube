package localinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var testSecrets = Secrets{RunnerKey: "tkckey_agent_RUNNERKEY", MasterPassword: "MASTERPW", MinioPassword: "MINIOPW", AIToken: "AITOKEN"}

const testLicense = "LICENSE-KEY"

type helmFake struct {
	status   string
	failOn   string
	failOut  string
	failures int
	calls    [][]string
	files    map[string]string
	manifest string
	cluster  string
	logs     string
	// Install blocks like a slow helm, until done or cancelled.
	blockFor time.Duration
	clusters []string
	reads    int
	mu       sync.Mutex
}

func (f *helmFake) run(ctx context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	for i, a := range args {
		if a == "-f" {
			data, _ := os.ReadFile(args[i+1])
			f.files[args[i+1][strings.LastIndex(args[i+1], string(os.PathSeparator))+1:]] = string(data)
		}
	}
	switch {
	case args[0] == "list" && f.status != "":
		return []byte(`[{"name":"x","status":"` + f.status + `"}]`), nil
	case args[0] == "list":
		return []byte(`[]`), nil
	case f.blockFor > 0 && args[0] == "upgrade" && args[2] == ReleaseName:
		select {
		case <-ctx.Done():
			return []byte("Release testkube has been cancelled.\nError: context canceled"), ctx.Err()
		case <-time.After(f.blockFor):
		}
	case f.failOn != "" && args[0] == "upgrade" && args[2] == f.failOn && f.failures != 0:
		f.failures--
		return []byte(f.failOut), errors.New("exit status 1")
	}
	return nil, nil
}

func newTestInstaller(t *testing.T, f *helmFake) *Installer {
	f.files = map[string]string{}
	return &Installer{helm: f.run, dir: t.TempDir(),
		wait: func(context.Context) ([]byte, error) {
			f.calls = append(f.calls, []string{"wait", "postgres"})
			return nil, nil
		},
		apply: func(_ context.Context, m string) ([]byte, error) { f.manifest = m; return nil, nil },
		read: func(context.Context) ([]byte, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.clusters) > 0 {
				c := f.clusters[min(f.reads, len(f.clusters)-1)]
				f.reads++
				return []byte(c), nil
			}
			if f.cluster == "" {
				return []byte(`{"items":[]}`), nil
			}
			return []byte(f.cluster), nil
		},
		logs: func(context.Context, string, bool) ([]byte, error) { return []byte(f.logs), nil }}
}

func (f *helmFake) commands() []string {
	var out []string
	for _, c := range f.calls {
		out = append(out, c[0]+" "+c[1])
	}
	return out
}

func TestInstall_StuckReleaseIsCleanedUpFirst(t *testing.T) {
	tests := map[string]struct {
		status        string
		wantRecovered bool
	}{
		"killed mid-install":  {"pending-install", true},
		"killed mid-upgrade":  {"pending-upgrade", true},
		"failed, retry works": {"failed", false},
		"installed before":    {"deployed", false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{status: tt.status}

			state, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

			require.NoError(t, err)
			assert.Equal(t, tt.wantRecovered, state.Recovered)
			assert.Equal(t, tt.wantRecovered, slices.Contains(f.commands(), "uninstall testkube"))
		})
	}
}

func TestInstall_SecretsStayOffTheCommandLineAndDisk(t *testing.T) {
	f := &helmFake{}
	i := newTestInstaller(t, f)

	_, _, err := i.Install(context.Background(), movedPorts, testSecrets, testLicense)
	require.NoError(t, err)

	for _, call := range f.calls {
		joined := strings.Join(call, " ")
		for _, secret := range []string{testLicense, testSecrets.RunnerKey, testSecrets.MasterPassword} {
			assert.NotContains(t, joined, secret)
		}
	}
	assert.Contains(t, f.manifest, `password: "MASTERPW"`)
	left, _ := os.ReadDir(i.dir)
	assert.Empty(t, left, "values files with the license are removed")
}

func TestInstall_LicenseFilesNeverOutliveTheInstall(t *testing.T) {
	tests := map[string]struct {
		failures int
		leftover bool
	}{
		"helm fails":                {failures: -1},
		"an earlier run was killed": {leftover: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{failOn: ReleaseName, failOut: "Error: boom", failures: tt.failures}
			i := newTestInstaller(t, f)
			if tt.leftover {
				require.NoError(t, os.MkdirAll(filepath.Join(i.dir, "install-123", "private.yaml"), 0o700))
			}

			_, _, _ = i.Install(context.Background(), movedPorts, testSecrets, testLicense)

			left, _ := os.ReadDir(i.dir)
			assert.Empty(t, left)
		})
	}
}

func TestInstall_RunnerKeyReachesBothCharts(t *testing.T) {
	f := &helmFake{}

	_, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)
	require.NoError(t, err)

	var private map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(f.files["private.yaml"]), &private))
	orgs := dig(private, "testkube-cloud-api", "api", "features", "bootstrapConfig", "config", "organizations").([]any)
	runner := dig(orgs[0], "runners").([]any)[0].(map[string]any)
	assert.Equal(t, testSecrets.RunnerKey, runner["secret_key"])
	assert.Equal(t, "demo-runner", runner["id"], "the rest of the bootstrap runner is kept")
	assert.Equal(t, testLicense, dig(private, "global", "enterpriseLicenseKey"))
	assert.Contains(t, f.files["runner.yaml"], "secret: "+testSecrets.RunnerKey)
}

func TestInstall_WorkerServiceUsesTheAPIsPostgres(t *testing.T) {
	f := &helmFake{}

	_, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)
	require.NoError(t, err)

	var worker, demo map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(f.files["demo-fixes.yaml"]), &worker))
	require.NoError(t, yaml.Unmarshal(EnterpriseDemoValues, &demo))
	assert.Equal(t, false, dig(worker, "testkube-worker-service", "api", "mongo", "enabled"))
	assert.Equal(t, true, dig(worker, "testkube-worker-service", "api", "postgres", "enabled"))
	assert.Equal(t, dig(demo, "testkube-cloud-api", "api", "postgres", "dsn"),
		dig(worker, "testkube-worker-service", "api", "postgres", "dsn"))
}

func TestInstall_TimeoutNamesTheStuckServiceAndItsLastLogs(t *testing.T) {
	crashing := `{"items":[{"kind":"Pod","metadata":{"name":"dex-1","labels":{"app.kubernetes.io/name":"dex"}},` +
		`"status":{"phase":"Running","containerStatuses":[{"name":"dex","restartCount":4,` +
		`"lastState":{"terminated":{"reason":"Error","exitCode":1,"finishedAt":"2999-01-01T00:00:00Z"}}}]}}]}`
	f := &helmFake{failOn: ReleaseName, failOut: "Error: context deadline exceeded", failures: 1, cluster: crashing,
		logs: `{"level":"fatal","msg":"failed to load config","error":"open /etc/dex/cfg/config.yaml: no such file"}`}

	_, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

	assert.ErrorIs(t, err, ErrInstallTimeout)
	var stuck StuckError
	require.ErrorAs(t, err, &stuck)
	assert.Equal(t, "dex", stuck.Service)
	assert.Equal(t, "crashloop", stuck.Reason)
	assert.Equal(t, []string{"failed to load config: open /etc/dex/cfg/config.yaml: no such file"}, stuck.Logs)
}

const (
	noMemoryCluster = `{"items":[{"kind":"Pod","metadata":{"name":"minio-1","labels":{"app.kubernetes.io/name":"minio"}},` +
		`"status":{"phase":"Pending","conditions":[{"type":"PodScheduled","status":"False","reason":"Unschedulable",` +
		`"message":"0/1 nodes are available: 1 Insufficient memory."}]}}]}`
	crashingCluster = `{"items":[{"kind":"Pod","metadata":{"name":"dex-1","labels":{"app.kubernetes.io/name":"dex"}},` +
		`"status":{"phase":"Running","containerStatuses":[{"name":"dex","restartCount":4,` +
		`"lastState":{"terminated":{"reason":"Error","exitCode":1,"finishedAt":"2999-01-01T00:00:00Z"}}}]}}]}`
)

func fastStuckWatch(t *testing.T) {
	poll, lasts := stuckPoll, stuckLasts
	stuckPoll, stuckLasts = 5*time.Millisecond, 150*time.Millisecond
	t.Cleanup(func() { stuckPoll, stuckLasts = poll, lasts })
}

func TestInstall_StopsEarlyOnlyWhenWaitingCantHelp(t *testing.T) {
	fastStuckWatch(t)
	tests := map[string]struct {
		clusters []string
		stopped  bool
	}{
		"not enough memory for a minute": {clusters: []string{noMemoryCluster}, stopped: true},
		"not enough memory that clears":  {clusters: []string{noMemoryCluster, noMemoryCluster, `{"items":[]}`}},
		"a crash may still recover":      {clusters: []string{crashingCluster}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{blockFor: time.Second, clusters: tt.clusters}

			_, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

			if !tt.stopped {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrStuck)
			assert.ErrorIs(t, err, ErrTestkubeInstall)
			var stuck StuckError
			require.ErrorAs(t, err, &stuck)
			assert.Equal(t, "no_memory", stuck.Reason)
			assert.Less(t, stuck.After, time.Second, "helm was stopped, not waited out")
			for _, call := range f.calls {
				assert.NotContains(t, call, runnerRelease, "the runner never starts")
			}
		})
	}
}

func TestInstall_FailuresNameTheirStage(t *testing.T) {
	tests := map[string]struct {
		failOn, out string
		want        []error
	}{
		"testkube not ready": {ReleaseName, "Error: context deadline exceeded", []error{ErrTestkubeInstall, ErrInstallTimeout}},
		"chart unreachable": {ReleaseName, "Error: failed to do request: dial tcp: lookup us-east1-docker.pkg.dev: no such host",
			[]error{ErrTestkubeInstall, ErrChartDownload}},
		"runner broken": {runnerRelease, "Error: unable to build kubernetes objects", []error{ErrRunnerInstall}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{failOn: tt.failOn, failOut: tt.out, failures: -1}

			_, out, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

			for _, want := range tt.want {
				assert.ErrorIs(t, err, want)
			}
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestInstall_MigrationThatLostTheRaceIsRetriedOnce(t *testing.T) {
	lost := "Error: resource Job/testkube/testkube-enterprise-api-migration-1 not ready. status: Failed, message: Job Failed. failed: 4/1"
	tests := map[string]struct {
		out          string
		failures     int
		wantAttempts int
		wantErr      bool
	}{
		"lost once, retry works":  {lost, 1, 2, false},
		"lost twice, gives up":    {lost, 2, 2, true},
		"other failure, no retry": {"Error: context deadline exceeded", 1, 1, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{failOn: ReleaseName, failOut: tt.out, failures: tt.failures}

			_, _, err := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

			assert.Equal(t, tt.wantErr, err != nil)
			attempts := 0
			for _, c := range f.calls {
				if c[0] == "upgrade" && c[2] == ReleaseName {
					attempts++
				}
			}
			assert.Equal(t, tt.wantAttempts, attempts)
			assert.Equal(t, tt.wantAttempts == 2, slices.Contains(f.commands(), "wait postgres"), "waits for postgres before retrying")
		})
	}
}
