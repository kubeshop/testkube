package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

func TestDiffDeploymentsReplicas(t *testing.T) {
	replicas := func(n int32) *int32 { return &n }
	deployment := func(r *int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
			Spec:       appsv1.DeploymentSpec{Replicas: r},
		}
	}

	for _, tc := range []struct {
		name      string
		old, new  *int32
		wantScale bool
	}{
		{name: "both unset", old: nil, new: nil},
		{name: "unset and explicit default are equal", old: nil, new: replicas(1)},
		{name: "explicit default and unset are equal", old: replicas(1), new: nil},
		{name: "different replica counts", old: replicas(1), new: replicas(2), wantScale: true},
		{name: "unset and non-default", old: nil, new: replicas(2), wantScale: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			causes := diffDeployments(deployment(tc.old), deployment(tc.new))
			gotScale := false
			for _, c := range causes {
				if c == testtrigger.CauseDeploymentScaleUpdate {
					gotScale = true
				}
			}
			assert.Equal(t, tc.wantScale, gotScale, "causes: %v", causes)
		})
	}
}
