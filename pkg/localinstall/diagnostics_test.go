package localinstall

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReport_NeverCopiesPodSettings(t *testing.T) {
	api := testPod("api", crashed(3, "Error", installEnd))
	api.Spec.Containers = []corev1.Container{{Name: "api", Args: []string{"--token=canary-arg-7f3a"},
		Env: []corev1.EnvVar{{Name: "ENTERPRISE_LICENSE_KEY", Value: "canary-license-91c2"}}}}

	var out strings.Builder
	buildReport(clusterSnapshot{pods: []corev1.Pod{api}}, nil).write(&out)

	for _, leak := range []string{"ENTERPRISE_LICENSE_KEY", "canary-license-91c2", "canary-arg-7f3a"} {
		assert.NotContains(t, out.String(), leak)
	}
	assert.Contains(t, out.String(), "api-pod  phase=Running ready=false restarts=3", "status still shows")
}

func TestReport_KeepsOnlyTheNewestEvents(t *testing.T) {
	var events []corev1.Event
	for i := range maxReportEvents + 5 {
		e := testEvent("Pod", "api-pod", "BackOff", fmt.Sprintf("event-%03d", i))
		e.LastTimestamp = metav1.NewTime(installStart.Add(time.Duration(i) * time.Second))
		events = append(events, e)
	}

	r := buildReport(clusterSnapshot{events: events}, nil)

	assert.Len(t, r.events, maxReportEvents)
	assert.Equal(t, fmt.Sprintf("event-%03d", maxReportEvents+4), r.events[0].message, "newest first")
}

func TestReport_PlantedSecretsNeverSurvive(t *testing.T) {
	license, runnerKey, password := "CANARY-LICENSE-1234-V3", "tkckey_agent_canary0123456789", "canary-p@ss/word=9"
	api := testPod("api", waitingFor("CreateContainerConfigError"), func(p *corev1.Pod) {
		p.Status.ContainerStatuses[0].State.Waiting.Message = "bad license " + license
	})
	events := []corev1.Event{
		testEvent("Pod", "api-pod", "Failed", "key="+url.QueryEscape(password)),
		testEvent("Pod", "api-pod", "Failed", "encoded "+base64.StdEncoding.EncodeToString([]byte(runnerKey))),
	}
	r := buildReport(clusterSnapshot{pods: []corev1.Pod{api}, events: events}, nil)
	r.logsByPod["api-pod"] = []string{"connect postgresql://testkube:" + password + "@db:5432/backend failed", "agent " + runnerKey}

	out := renderReport(r, newRedactor(license, runnerKey, password))

	for _, v := range []string{license, runnerKey, password, url.QueryEscape(password), base64.StdEncoding.EncodeToString([]byte(runnerKey))} {
		assert.NotContains(t, out, v)
	}
	assert.Contains(t, out, "postgresql://testkube:[removed]@db:5432", "the address stays readable")
}

func TestRedactor_PatternsCatchUnknownSecretsButSpareTheRest(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"dsn password":           {"dial postgresql://u:s3cr@t@db:5432/x", "dial postgresql://u:[removed]@db:5432/x"},
		"dsn without password":   {"dial postgresql://db:5432/x", "dial postgresql://db:5432/x"},
		"bearer token":           {"Authorization: Bearer abc.def-123", "Authorization: Bearer [removed]"},
		"key value":              {`api_key="zzz123" next`, `api_key="[removed]" next`},
		"short known value kept": {"login with password", "login with password"},
		"api key header":         {"X-Api-Key: abc123def", "X-Api-Key: [removed]"},
		"basic auth":             {"Authorization: Basic dXNlcjpwYXNz", "Authorization: Basic [removed]"},
		"license flag":           {"run --license ABCD-1234-EFGH", "run --license [removed]"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, newRedactor("password").clean(tt.in))
		})
	}
}

func TestSaveReport_OnlyAfterARealFailure(t *testing.T) {
	leaky := `{"items":[{"kind":"Pod","metadata":{"name":"dex-1","labels":{"app.kubernetes.io/name":"dex"}},` +
		`"status":{"phase":"Running"}}]}`
	tests := map[string]struct {
		cancelled bool
		saved     bool
	}{
		"helm timed out": {saved: true},
		"user quit":      {cancelled: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := &helmFake{failOn: ReleaseName, failures: 1, failOut: "Error: context deadline exceeded", cluster: leaky,
				logs: "connecting with " + testSecrets.RunnerKey}
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelled {
				cancel()
			}
			defer cancel()
			i := newTestInstaller(t, f)

			state, _, _ := i.Install(ctx, movedPorts, testSecrets, testLicense)

			files, _ := filepath.Glob(filepath.Join(i.dir, "logs", "install-*.txt"))
			if !tt.saved {
				assert.Empty(t, files)
				assert.Empty(t, state.ReportPath)
				return
			}
			require.Len(t, files, 1)
			assert.Equal(t, files[0], state.ReportPath)
			data, err := os.ReadFile(files[0])
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(string(data), "Testkube install report, saved after a failed install on "))
			assert.Contains(t, string(data), "dex-1", "the stuck pod's state is there")
			assert.Contains(t, string(data), "CLI: v9.9.9")
			assert.Contains(t, string(data), "Docker: Docker Desktop 29.0.0, 8 CPUs, 8.0 GB memory")
			assert.Contains(t, string(data), "Ports: dashboard 8081, api 8091, login 5557, storage 9001, ai 9091")
			assert.NotContains(t, string(data), testSecrets.RunnerKey)
			if runtime.GOOS != "windows" {
				info, err := os.Stat(files[0])
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}
		})
	}
}

func TestSaveReport_KeepsTheNewestTenAndTheLastLogLines(t *testing.T) {
	var lines []string
	for n := range 1000 {
		lines = append(lines, fmt.Sprintf("line-%04d", n))
	}
	f := &helmFake{failOn: ReleaseName, failures: 1, failOut: "Error: context deadline exceeded",
		cluster: `{"items":[{"kind":"Pod","metadata":{"name":"dex-1"},"status":{"phase":"Running"}}]}`,
		logs:    strings.Join(lines, "\n")}
	i := newTestInstaller(t, f)
	logs := filepath.Join(i.dir, "logs")
	require.NoError(t, os.MkdirAll(logs, 0o700))
	for n := range keptReports + 2 {
		require.NoError(t, os.WriteFile(filepath.Join(logs, fmt.Sprintf("install-20250101-0000%02d.txt", n)), nil, 0o600))
	}

	state, _, _ := i.Install(context.Background(), movedPorts, testSecrets, testLicense)

	files, _ := filepath.Glob(filepath.Join(logs, "install-*.txt"))
	assert.Len(t, files, keptReports)
	assert.NotContains(t, files, filepath.Join(logs, "install-20250101-000000.txt"), "the oldest went")
	assert.Contains(t, files, state.ReportPath, "the new one stays")
	data, err := os.ReadFile(state.ReportPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "line-0999")
	assert.NotContains(t, string(data), fmt.Sprintf("line-%04d", 999-reportLogLines), "only the last lines")
}

func TestCapReport_StopsAtOneMegabyte(t *testing.T) {
	assert.LessOrEqual(t, len(capReport(strings.Repeat("x", 3*maxReportBytes))), maxReportBytes+32)
	assert.Equal(t, "short", capReport("short"))
}

func TestSaveReport_SecretAcrossALongLineCutIsStillMasked(t *testing.T) {
	f := &helmFake{failOn: ReleaseName, failures: 1, failOut: "Error: context deadline exceeded",
		cluster: `{"items":[{"kind":"Pod","metadata":{"name":"dex-1"},"status":{"phase":"Running"}}]}`,
		logs:    strings.Repeat("x", 2040) + testSecrets.RunnerKey}

	state, _, _ := newTestInstaller(t, f).Install(context.Background(), movedPorts, testSecrets, testLicense)

	data, err := os.ReadFile(state.ReportPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), testSecrets.RunnerKey[:8], "not even the part before the cut")
}

func TestDemoSecrets_ComeFromTheEmbeddedValues(t *testing.T) {
	assert.ElementsMatch(t, []string{"postgres5432", "QWkVzs3nct6HZM5hxsPzwaZtq"}, demoSecrets())
}

func TestCutAt_NeverSplitsACharacter(t *testing.T) {
	cut := cutAt(strings.Repeat("é", 10), 5)

	assert.True(t, utf8.ValidString(cut))
	assert.Equal(t, "éé", cut)
}

func TestReport_AFailedJobShowsItsRealExitCode(t *testing.T) {
	job := testPod("migration", func(p *corev1.Pod) {
		p.Status.Phase = corev1.PodFailed
		p.Status.ContainerStatuses[0].State.Terminated = &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 3}
	})

	var out strings.Builder
	buildReport(clusterSnapshot{pods: []corev1.Pod{job}}, nil).write(&out)

	assert.Contains(t, out.String(), "exit=3")
	assert.Contains(t, out.String(), "    Error\n")
}
