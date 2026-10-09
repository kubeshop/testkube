package localinstall

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	r.logs["api-pod"] = []string{"connect postgresql://testkube:" + password + "@db:5432/backend failed", "agent " + runnerKey}

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
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, newRedactor("password").clean(tt.in))
		})
	}
}
