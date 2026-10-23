package v1

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

func httpContentSpec(mutate func(*TestTriggerSpec)) TestTriggerSpec {
	spec := TestTriggerSpec{
		Resource: TestTriggerResourceContent,
		Event:    TestTriggerEventHTTPChanged,
		ContentSelector: &TestTriggerContentSelector{HTTP: &TestTriggerContentHTTPSpec{
			URL:    "https://api.example.com/version",
			Select: &TestTriggerContentHTTPSelect{JSONPath: ".data.version"},
		}},
	}
	if mutate != nil {
		mutate(&spec)
	}
	return spec
}

func TestTestTriggerSpecValidate_HTTPContent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate  func(*TestTriggerSpec)
		wantErr string
	}{
		"valid": {},
		"valid with every field": {mutate: func(s *TestTriggerSpec) {
			h := s.ContentSelector.HTTP
			h.Method = "POST"
			h.Headers = map[string]string{"Accept": "application/json"}
			h.HeadersFrom = []TestTriggerContentHTTPHeaderSource{{
				Name:      "Authorization",
				ValueFrom: corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{Key: "bearer"}},
			}}
			h.Interval = &metav1.Duration{Duration: time.Minute}
			h.Timeout = &metav1.Duration{Duration: 5 * time.Second}
			h.ExpectedStatus = []int32{200, 204}
		}},
		"match without a listener": {mutate: func(s *TestTriggerSpec) {
			s.Match = []workflowtriggersv1.WorkflowTriggerFieldCondition{{Path: ".value", Operator: workflowtriggersv1.FieldOperatorEquals, Value: "2.0.0"}}
		}},
		"a change operator": {mutate: func(s *TestTriggerSpec) {
			s.Match = []workflowtriggersv1.WorkflowTriggerFieldCondition{{Path: ".value", Operator: workflowtriggersv1.FieldOperatorChangedFrom, Value: "1.0.0"}}
		}},
		"a git event": {
			mutate:  func(s *TestTriggerSpec) { s.Event = TestTriggerEventGitPush },
			wantErr: `requires event to be "http-changed"`,
		},
		"both sources": {
			mutate: func(s *TestTriggerSpec) {
				s.ContentSelector.Git = &TestTriggerContentGitSpec{Uri: "https://github.com/kubeshop/testkube"}
			},
			wantErr: "only one of contentSelector.git or contentSelector.http",
		},
		"no URL": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.URL = "" },
			wantErr: "url is required",
		},
		"a URL that is not http": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.URL = "ftp://example.com/version" },
			wantErr: "absolute http or https URL",
		},
		"a relative URL": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.URL = "/version" },
			wantErr: "absolute http or https URL",
		},
		"an unknown method": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.Method = "get" },
			wantErr: "method",
		},
		"a short interval": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.Interval = &metav1.Duration{Duration: time.Second} },
			wantErr: "interval must be at least 10s",
		},
		"a zero timeout": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.Timeout = &metav1.Duration{} },
			wantErr: "timeout must be positive",
		},
		"a header source without a reference": {
			mutate: func(s *TestTriggerSpec) {
				s.ContentSelector.HTTP.HeadersFrom = []TestTriggerContentHTTPHeaderSource{{Name: "Authorization"}}
			},
			wantErr: "requires secretKeyRef or configMapKeyRef",
		},
		"a status that is not HTTP": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.ExpectedStatus = []int32{999} },
			wantErr: "999 is not an HTTP status code",
		},
		"two selections": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.Select.Header = "ETag" },
			wantErr: "only one of jsonPath or header",
		},
		"a JSONPath with an array index": {
			mutate:  func(s *TestTriggerSpec) { s.ContentSelector.HTTP.Select.JSONPath = ".items[0].version" },
			wantErr: "is not a valid dot-path",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spec := httpContentSpec(tt.mutate)
			errs := spec.Validate()

			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("expected no validation errors, got %v", errs)
				}
				return
			}
			for _, err := range errs {
				if strings.Contains(err.Error(), tt.wantErr) {
					return
				}
			}
			t.Fatalf("expected an error containing %q, got %v", tt.wantErr, errs)
		})
	}
}

func TestTestTriggerSpecValidate_GitContentStillRejectsHTTPChanged(t *testing.T) {
	t.Parallel()

	spec := TestTriggerSpec{
		Resource:        TestTriggerResourceContent,
		Event:           TestTriggerEventHTTPChanged,
		ContentSelector: &TestTriggerContentSelector{Git: &TestTriggerContentGitSpec{Uri: "https://github.com/kubeshop/testkube"}},
	}

	if errs := spec.Validate(); len(errs) == 0 {
		t.Fatalf("expected a validation error for a git trigger on http-changed")
	}
}
