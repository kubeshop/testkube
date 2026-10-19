// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// stubAPIServer answers DeleteCollection with a per-resource status, so the cleanup error is
// produced by real client-go from a real HTTP response rather than hand-constructed.
func stubAPIServer(t *testing.T, statusByResource map[string]int, bareBody bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resource := ""
		for _, candidate := range []string{"pods", "secrets", "configmaps", "persistentvolumeclaims", "jobs"} {
			if strings.Contains(r.URL.Path, "/"+candidate) {
				resource = candidate
				break
			}
		}
		code, ok := statusByResource[resource]
		if !ok {
			code = http.StatusOK
		}
		if code == http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Success","code":200}`)
			return
		}
		if bareBody {
			// What a load balancer in front of kube-apiserver sends: not a k8s Status at all.
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(code)
			fmt.Fprint(w, "<html><body>502 Bad Gateway</body></html>")
			return
		}
		reason := map[int]string{503: "ServiceUnavailable", 403: "Forbidden", 500: "InternalError", 429: "TooManyRequests"}[code]
		message := map[int]string{
			503: "the server is currently unable to handle the request",
			403: fmt.Sprintf("%s is forbidden: User \"system:serviceaccount:tk-dev:tests-job\" cannot deletecollection resource %q", resource, resource),
			500: "Internal error occurred: failed calling webhook \"validate.example.com\"",
			429: "too many requests",
		}[code]
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":%q,"reason":%q,"details":{"group":"","kind":%q},"code":%d}`, message, reason, resource, code)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func realClientSet(t *testing.T, host string) kubernetes.Interface {
	t.Helper()
	cs, err := kubernetes.NewForConfig(&rest.Config{Host: host})
	if err != nil {
		t.Fatalf("clientset: %v", err)
	}
	return cs
}

// The group cleanup always returns an errors.Join aggregate, even when a single delete failed, so
// the classifier's tree walk is the normal path rather than an edge case. These cases drive real
// client-go against a stub API server: the errors are produced from real HTTP responses, which is
// what the mock-based tests above cannot establish.
func TestRunKillOverTheWire(t *testing.T) {
	everyKind := map[string]int{"pods": 503, "secrets": 503, "configmaps": 503, "persistentvolumeclaims": 503, "jobs": 503}

	tests := []struct {
		name         string
		statuses     map[string]int
		bareBody     bool
		wantAttempts int
		wantWarning  bool
		wantErr      bool
		wantErrParts []string
	}{
		{
			name:     "every delete returns 503: retried, downgraded, step passes",
			statuses: everyKind, wantAttempts: cleanupRetryCount, wantWarning: true,
		},
		{
			name:     "one delete returns 503: retried, downgraded, step passes",
			statuses: map[string]int{"pods": 503}, wantAttempts: cleanupRetryCount, wantWarning: true,
		},
		{
			// A load balancer in front of the API server answers with HTML, not a k8s Status. The
			// reason-based helpers cannot see it, so it is matched by status code.
			name:     "bare 502 from a load balancer: retried, downgraded, step passes",
			statuses: map[string]int{"pods": 502}, bareBody: true,
			wantAttempts: cleanupRetryCount, wantWarning: true,
		},
		{
			// The reported incident's shape: a draining Envoy sidecar answers 503 with its own
			// body rather than a k8s Status. client-go renders this as the exact message the
			// customer saw, "the server is currently unable to handle the request (delete pods)".
			name:     "bare 503 from a draining sidecar: retried, downgraded, step passes",
			statuses: map[string]int{"pods": 503}, bareBody: true,
			wantAttempts: cleanupRetryCount, wantWarning: true,
		},
		{
			name:     "403 is permanent: one attempt, step fails",
			statuses: map[string]int{"pods": 403}, wantAttempts: 1, wantErr: true,
			wantErrParts: []string{"cleaning up resources", "is forbidden"},
		},
		{
			// A failing admission webhook surfaces as a 500 and recurs on every run.
			name:     "500 from a webhook is permanent: one attempt, step fails",
			statuses: map[string]int{"pods": 500}, wantAttempts: 1, wantErr: true,
			wantErrParts: []string{"cleaning up resources", "failed calling webhook"},
		},
		{
			// The decisive case: the transient branch earns the retries, but the permanent branch
			// still fails the step. Downgrading here would report a missing RBAC rule only on the
			// runs where no blip happened to co-occur with it.
			name:         "mixed 503 and 403: retried, still fails, both causes reported",
			statuses:     map[string]int{"pods": 503, "secrets": 403},
			wantAttempts: cleanupRetryCount, wantErr: true,
			wantErrParts: []string{"the server is currently unable to handle the request", "secrets is forbidden"},
		},
		{
			name:     "everything deletes: one attempt, step passes",
			statuses: map[string]int{}, wantAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortenCleanupRetryDelay(t)
			srv := stubAPIServer(t, tt.statuses, tt.bareBody)
			cs := realClientSet(t, srv.URL)

			ctrl := gomock.NewController(t)
			worker := executionworkertypes.NewMockWorker(ctrl)
			worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

			attempts := 0
			worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
				DoAndReturn(func(c context.Context, groupID string, _ executionworkertypes.DestroyOptions) error {
					attempts++
					return controller.CleanupGroup(c, cs, "tk-dev", groupID)
				}).AnyTimes()

			var err error
			out := captureStdout(t, func() {
				err = RunKill(context.Background(), worker, healthyClientSet(), "tk-dev", "ref", "svc-group", nil, nil)
			})

			assert.Equal(t, tt.wantAttempts, attempts, "attempts")
			assert.Equal(t, tt.wantWarning, strings.Contains(out, "warning: cleaning up resources"), "warning emitted")
			if tt.wantErr {
				require.Error(t, err)
				for _, part := range tt.wantErrParts {
					assert.Contains(t, err.Error(), part)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Once the sidecar is gone there is nothing to answer at all, so the delete never reaches the API
// server. That is still the test's result rather than its verdict.
func TestRunKillWhenTheSidecarIsGone(t *testing.T) {
	shortenCleanupRetryDelay(t)

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	cs := realClientSet(t, url)

	ctrl := gomock.NewController(t)
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

	attempts := 0
	worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
		DoAndReturn(func(c context.Context, groupID string, _ executionworkertypes.DestroyOptions) error {
			attempts++
			return controller.CleanupGroup(c, cs, "tk-dev", groupID)
		}).AnyTimes()

	var err error
	out := captureStdout(t, func() {
		err = RunKill(context.Background(), worker, healthyClientSet(), "tk-dev", "ref", "svc-group", nil, nil)
	})

	assert.NoError(t, err)
	assert.Equal(t, cleanupRetryCount, attempts)
	assert.Contains(t, out, "warning: cleaning up resources")
	assert.Contains(t, out, "connection refused")
}
