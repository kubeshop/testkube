package k8s

import "testing"

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
