package testtriggers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestContentHTTP_RoundTrips(t *testing.T) {
	crd := &testsv1.TestTriggerContentSelector{HTTP: &testsv1.TestTriggerContentHTTPSpec{
		URL:     "https://api.example.com/version",
		Method:  "POST",
		Headers: map[string]string{"Accept": "application/json"},
		HeadersFrom: []testsv1.TestTriggerContentHTTPHeaderSource{{
			Name: "Authorization",
			ValueFrom: corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-token"},
				Key:                  "bearer",
			}},
		}},
		Body:                   `{"query": "{ version }"}`,
		Interval:               &metav1.Duration{Duration: time.Minute},
		Timeout:                &metav1.Duration{Duration: 5 * time.Second},
		InsecureSkipVerify:     true,
		ExpectedStatus:         []int32{200, 204},
		Select:                 &testsv1.TestTriggerContentHTTPSelect{JSONPath: ".data.version"},
		FireOnFirstObservation: true,
	}}

	api := mapContentSelectorFromCRD(crd)

	assert.Equal(t, "1m0s", api.Http.Interval)
	assert.Equal(t, ".data.version", api.Http.Select_.JsonPath)
	assert.Equal(t, crd, mapContentSelectorToCRD(api))
}

// An unparsable duration must reach validation, which rejects it, rather than vanish.
func TestContentHTTP_KeepsAnUnparsableDurationForValidation(t *testing.T) {
	crd := mapContentHTTPToCRD(&testkube.TestTriggerContentHttp{Url: "https://api.example.com", Interval: "every minute"})

	assert.NotNil(t, crd.Interval)
	assert.NotEmpty(t, crd.Validate())
}
