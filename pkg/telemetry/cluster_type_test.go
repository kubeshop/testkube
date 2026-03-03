package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDetectFromProviderID(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		want       string
	}{
		{"GKE", "gce://my-project/us-central1-a/node-1", "gke"},
		{"EKS", "aws:///us-east-1a/i-0123456789abcdef", "eks"},
		{"AKS", "azure:///subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm-1", "aks"},
		{"Kind", "kind://docker/kind/kind-control-plane", "kind"},
		{"DigitalOcean", "digitalocean://12345", "doks"},
		{"OpenStack", "openstack:///instance-id", "openstack"},
		{"case insensitive", "GCE://my-project/us-central1-a/node-1", "gke"},
		{"empty providerID", "", ""},
		{"unknown provider", "vmware://datacenter/vm-1", ""},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Spec:       corev1.NodeSpec{ProviderID: tt.providerID},
			})
			assert.Equal(t, tt.want, detectFromProviderID(ctx, cs))
		})
	}
}

func TestDetectFromNodeLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{"EKS label", map[string]string{"eks.amazonaws.com/nodegroup": "ng-1"}, "eks"},
		{"GKE label", map[string]string{"cloud.google.com/gke-nodepool": "pool-1"}, "gke"},
		{"AKS label", map[string]string{"kubernetes.azure.com/cluster": "my-cluster"}, "aks"},
		{"Minikube label", map[string]string{"minikube.k8s.io/name": "minikube"}, "minikube"},
		{"OpenShift label", map[string]string{"node.openshift.io/os_id": "rhcos"}, "openshift"},
		{"DigitalOcean label", map[string]string{"doks.digitalocean.com/node-id": "abc"}, "doks"},
		{"MicroK8s label", map[string]string{"microk8s.io/cluster": "default"}, "microk8s"},
		{"no matching labels", map[string]string{"kubernetes.io/os": "linux"}, ""},
		{"nil labels", nil, ""},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1", Labels: tt.labels},
			})
			assert.Equal(t, tt.want, detectFromNodeLabels(ctx, cs))
		})
	}
}

func TestDetectFromServerVersion(t *testing.T) {
	cs := fake.NewSimpleClientset()
	assert.Equal(t, "", detectFromServerVersion(context.Background(), cs))
}

func TestDetectFromKubeSystemPods(t *testing.T) {
	tests := []struct {
		name     string
		podNames []string
		want     string
	}{
		{"Kind via kindnet", []string{"kindnet-abc12"}, "kind"},
		{"Kind via -kind-", []string{"kube-proxy-kind-abc"}, "kind"},
		{"Minikube", []string{"kube-proxy-minikube"}, "minikube"},
		{"Docker Desktop", []string{"vpnkit-docker-desktop"}, "docker-desktop"},
		{"GKE", []string{"gke-metrics-agent-xyz"}, "gke"},
		{"EKS", []string{"aws-node-abc12"}, "eks"},
		{"AKS", []string{"azure-ip-masq-agent-xyz"}, "aks"},
		{"AKS azuredisk", []string{"csi-azuredisk-node-xyz"}, "aks"},
		{"OpenShift", []string{"openshift-apiserver-abc"}, "openshift"},
		{"k3d", []string{"k3d-proxy-abc"}, "k3d"},
		{"k3s", []string{"k3s-server-abc"}, "k3s"},
		{"MicroK8s", []string{"microk8s-dashboard-abc"}, "microk8s"},
		{"no match", []string{"coredns-abc", "etcd-master"}, ""},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := buildFakeClientWithPods(tt.podNames)
			assert.Equal(t, tt.want, detectFromKubeSystemPods(ctx, cs))
		})
	}
}

func buildFakeClientWithPods(podNames []string) kubernetes.Interface {
	objs := make([]runtime.Object, 0, len(podNames))
	for _, name := range podNames {
		objs = append(objs, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "kube-system"},
		})
	}
	return fake.NewSimpleClientset(objs...)
}

func TestDetectClusterTypeFromClientset_LayerPriority(t *testing.T) {
	t.Run("providerID takes priority over pod names", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Spec:       corev1.NodeSpec{ProviderID: "gce://project/zone/node-1"},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "aws-node-abc", Namespace: "kube-system"},
			},
		)
		assert.Equal(t, "gke", detectClusterTypeFromClientset(cs))
	})

	t.Run("node labels used when providerID absent", func(t *testing.T) {
		cs := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "node-1",
				Labels: map[string]string{"eks.amazonaws.com/nodegroup": "ng-1"},
			},
		})
		assert.Equal(t, "eks", detectClusterTypeFromClientset(cs))
	})

	t.Run("pod names used as last resort", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "kindnet-abc", Namespace: "kube-system"},
			},
		)
		assert.Equal(t, "kind", detectClusterTypeFromClientset(cs))
	})

	t.Run("returns others when nothing matches", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "coredns-abc", Namespace: "kube-system"},
			},
		)
		assert.Equal(t, "others", detectClusterTypeFromClientset(cs))
	})
}
