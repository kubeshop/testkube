package testtriggers

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
)

func mapContentHTTPFromCRD(h *testsv1.TestTriggerContentHTTPSpec) *testkube.TestTriggerContentHttp {
	if h == nil {
		return nil
	}
	result := &testkube.TestTriggerContentHttp{
		Url:                    h.URL,
		Method:                 h.Method,
		Headers:                h.Headers,
		Body:                   h.Body,
		Interval:               durationString(h.Interval),
		Timeout:                durationString(h.Timeout),
		InsecureSkipVerify:     h.InsecureSkipVerify,
		ExpectedStatus:         h.ExpectedStatus,
		FireOnFirstObservation: h.FireOnFirstObservation,
	}
	for _, header := range h.HeadersFrom {
		result.HeadersFrom = append(result.HeadersFrom, testkube.TestTriggerContentHttpHeaderSource{
			Name:      header.Name,
			ValueFrom: commonmapper.MapEnvVarSourceKubeToAPI(&header.ValueFrom),
		})
	}
	if h.Select != nil {
		result.Select_ = &testkube.TestTriggerContentHttpSelect{JsonPath: h.Select.JSONPath, Header: h.Select.Header}
	}
	return result
}

func mapContentHTTPToCRD(h *testkube.TestTriggerContentHttp) *testsv1.TestTriggerContentHTTPSpec {
	if h == nil {
		return nil
	}
	result := &testsv1.TestTriggerContentHTTPSpec{
		URL:                    h.Url,
		Method:                 h.Method,
		Headers:                h.Headers,
		Body:                   h.Body,
		Interval:               parseDuration(h.Interval),
		Timeout:                parseDuration(h.Timeout),
		InsecureSkipVerify:     h.InsecureSkipVerify,
		ExpectedStatus:         h.ExpectedStatus,
		FireOnFirstObservation: h.FireOnFirstObservation,
	}
	for _, header := range h.HeadersFrom {
		source := corev1.EnvVarSource{}
		if mapped := commonmapper.MapEnvVarSourceAPIToKube(header.ValueFrom); mapped != nil {
			source = *mapped
		}
		result.HeadersFrom = append(result.HeadersFrom, testsv1.TestTriggerContentHTTPHeaderSource{Name: header.Name, ValueFrom: source})
	}
	if h.Select_ != nil {
		result.Select = &testsv1.TestTriggerContentHTTPSelect{JSONPath: h.Select_.JsonPath, Header: h.Select_.Header}
	}
	return result
}

func durationString(d *metav1.Duration) string {
	if d == nil {
		return ""
	}
	return d.Duration.String()
}

// parseDuration maps an unparsable duration to zero rather than dropping it, so that validation
// rejects it instead of the trigger silently falling back to the default.
func parseDuration(s string) *metav1.Duration {
	if s == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return &metav1.Duration{}
	}
	return &metav1.Duration{Duration: d}
}
