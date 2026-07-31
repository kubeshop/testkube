package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestLeaseName_Prefix(t *testing.T) {
	b := NewK8sLeaseBackend(nil, "testkube-triggers-lease", "")

	name := b.leaseName("cluster-a")

	if name != "testkube-triggers-lease-cluster-a" {
		t.Fatalf("expected default lease name 'testkube-triggers-lease-cluster-a', got %q", name)
	}
}

func TestLeaseName_Override(t *testing.T) {
	b := NewK8sLeaseBackend(nil, "lease-prefix", "", WithLeaseName("custom-lease"))

	name := b.leaseName("cluster-b")

	if name != "custom-lease" {
		t.Fatalf("expected custom lease name 'custom-lease', got %q", name)
	}
}

func TestK8sLeaseBackend_TryAcquire(t *testing.T) {
	const (
		namespace = "testkube"
		leaseName = "testkube-lease"
		holder    = "instance-1"
		other     = "instance-2"
	)

	conflict := apierrors.NewConflict(
		schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"},
		leaseName,
		errors.New("object was modified"),
	)
	internal := apierrors.NewInternalError(errors.New("api server unavailable"))

	lease := func(holderIdentity string, renewedAgo time.Duration) *coordv1.Lease {
		renewTime := metav1.MicroTime{Time: time.Now().Add(-renewedAgo)}
		durationSeconds := int32(60)
		return &coordv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: leaseName, Namespace: namespace},
			Spec: coordv1.LeaseSpec{
				HolderIdentity:       &holderIdentity,
				AcquireTime:          &renewTime,
				RenewTime:            &renewTime,
				LeaseDurationSeconds: &durationSeconds,
			},
		}
	}

	tests := []struct {
		name string
		// lease is the object already present in the cluster, nil when it does not exist yet.
		lease *coordv1.Lease
		// updateErrs is indexed by update call; updates past the last entry succeed.
		updateErrs []error
		// stolenOnReread makes every Get after the first report another holder.
		stolenOnReread bool
		wantLeased     bool
		wantErr        bool
		wantUpdates    int
	}{
		{
			name:        "creates the lease when it does not exist",
			wantLeased:  true,
			wantUpdates: 0,
		},
		{
			name:        "renews a lease we already hold",
			lease:       lease(holder, 10*time.Second),
			wantLeased:  true,
			wantUpdates: 1,
		},
		{
			name:        "retries the renewal once when the update conflicts",
			lease:       lease(holder, 10*time.Second),
			updateErrs:  []error{conflict},
			wantLeased:  true,
			wantUpdates: 2,
		},
		{
			name:        "gives up the lease when renewal conflicts persist",
			lease:       lease(holder, 10*time.Second),
			updateErrs:  []error{conflict, conflict},
			wantLeased:  false,
			wantUpdates: 2,
		},
		{
			name:           "reports not leased when another instance took over during a renewal conflict",
			lease:          lease(holder, 10*time.Second),
			updateErrs:     []error{conflict},
			stolenOnReread: true,
			wantLeased:     false,
			wantUpdates:    1,
		},
		{
			name:        "propagates non-conflict renewal errors",
			lease:       lease(holder, 10*time.Second),
			updateErrs:  []error{internal},
			wantLeased:  false,
			wantErr:     true,
			wantUpdates: 1,
		},
		{
			name:        "takes over an expired lease",
			lease:       lease(other, 2*time.Minute),
			wantLeased:  true,
			wantUpdates: 1,
		},
		{
			name:        "leaves a live lease to its holder",
			lease:       lease(other, 10*time.Second),
			wantLeased:  false,
			wantUpdates: 0,
		},
		{
			name:        "does not retry a conflicting takeover",
			lease:       lease(other, 2*time.Minute),
			updateErrs:  []error{conflict},
			wantLeased:  false,
			wantUpdates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []runtime.Object
			if tt.lease != nil {
				objects = append(objects, tt.lease)
			}

			clientset := fake.NewSimpleClientset(objects...)

			gets := 0
			clientset.PrependReactor("get", "leases", func(_ ktesting.Action) (bool, runtime.Object, error) {
				gets++
				if tt.stolenOnReread && gets > 1 {
					return true, lease(other, 10*time.Second), nil
				}
				return false, nil, nil
			})

			updates := 0
			clientset.PrependReactor("update", "leases", func(_ ktesting.Action) (bool, runtime.Object, error) {
				updates++
				if updates <= len(tt.updateErrs) {
					return true, nil, tt.updateErrs[updates-1]
				}
				return false, nil, nil
			})

			backend := NewK8sLeaseBackend(clientset, "testkube", namespace, WithLeaseName(leaseName))

			leased, err := backend.TryAcquire(context.Background(), holder, "cluster-1")

			if tt.wantErr != (err != nil) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if leased != tt.wantLeased {
				t.Fatalf("expected leased %v, got %v", tt.wantLeased, leased)
			}
			if updates != tt.wantUpdates {
				t.Fatalf("expected %d update calls, got %d", tt.wantUpdates, updates)
			}
		})
	}
}
