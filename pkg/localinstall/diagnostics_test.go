package localinstall

import (
	"fmt"
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
