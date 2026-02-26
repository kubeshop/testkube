package telemetry

import (
	"context"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
)

const clusterTypeDetectionTimeout = 10 * time.Second

var (
	clusterTypeOnce   sync.Once
	cachedClusterType string
)

// GetClusterType returns the detected Kubernetes cluster type.
// The result is cached after the first call since the cluster type
// does not change during the lifetime of the process.
func GetClusterType() string {
	clusterTypeOnce.Do(func() {
		cachedClusterType = detectClusterType()
	})
	return cachedClusterType
}

func detectClusterType() string {
	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		log.DefaultLogger.Debugw("cluster type detection: k8s connect failed", "error", err)
		return "unidentified"
	}
	return detectClusterTypeFromClientset(clientset)
}

// detectClusterTypeFromClientset runs a layered detection chain against the
// given clientset with a shared timeout so the entire detection is bounded.
func detectClusterTypeFromClientset(clientset kubernetes.Interface) string {
	ctx, cancel := context.WithTimeout(context.Background(), clusterTypeDetectionTimeout)
	defer cancel()

	detectors := []func(context.Context, kubernetes.Interface) string{
		detectFromProviderID,
		detectFromNodeLabels,
		detectFromServerVersion,
		detectFromKubeSystemPods,
	}

	for _, detect := range detectors {
		if ctx.Err() != nil {
			log.DefaultLogger.Debugw("cluster type detection: timeout reached before all layers checked")
			break
		}
		if ct := detect(ctx, clientset); ct != "" {
			return ct
		}
	}

	return "others"
}

// --- Layer 1: Node spec.providerID ---
// The kubelet / cloud-controller-manager populates this with a well-known
// URI scheme per provider, making it the most deterministic signal.

var providerIDPrefixes = []struct {
	prefix      string
	clusterType string
}{
	{"kind://", "kind"},
	{"gce://", "gke"},
	{"aws://", "eks"},
	{"azure://", "aks"},
	{"digitalocean://", "doks"},
	{"openstack://", "openstack"},
}

func detectFromProviderID(ctx context.Context, clientset kubernetes.Interface) string {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 5})
	if err != nil || len(nodes.Items) == 0 {
		return ""
	}

	for _, node := range nodes.Items {
		pid := strings.ToLower(node.Spec.ProviderID)
		if pid == "" {
			continue
		}
		for _, p := range providerIDPrefixes {
			if strings.HasPrefix(pid, p.prefix) {
				return p.clusterType
			}
		}
	}
	return ""
}

// --- Layer 2: Node labels ---
// Cloud-managed clusters inject provider-specific labels that are stable
// across versions. This also covers local distros that lack a providerID.

var labelDetectors = []struct {
	label       string
	clusterType string
}{
	{"eks.amazonaws.com/nodegroup", "eks"},
	{"cloud.google.com/gke-nodepool", "gke"},
	{"kubernetes.azure.com/cluster", "aks"},
	{"doks.digitalocean.com/node-id", "doks"},
	{"minikube.k8s.io/name", "minikube"},
	{"node.openshift.io/os_id", "openshift"},
	{"microk8s.io/cluster", "microk8s"},
}

func detectFromNodeLabels(ctx context.Context, clientset kubernetes.Interface) string {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 5})
	if err != nil || len(nodes.Items) == 0 {
		return ""
	}

	for _, node := range nodes.Items {
		for _, ld := range labelDetectors {
			if _, ok := node.Labels[ld.label]; ok {
				return ld.clusterType
			}
		}
	}
	return ""
}

// --- Layer 3: Kubernetes server version string ---
// GKE, EKS, k3s and others embed distribution identifiers in the API
// server's GitVersion.

func detectFromServerVersion(_ context.Context, clientset kubernetes.Interface) string {
	sv, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return ""
	}

	gitVersion := strings.ToLower(sv.GitVersion)

	switch {
	case strings.Contains(gitVersion, "-gke."):
		return "gke"
	case strings.Contains(gitVersion, "-eks-"):
		return "eks"
	case strings.Contains(gitVersion, "+k3s"):
		return "k3s"
	case strings.Contains(gitVersion, "+k0s"):
		return "k0s"
	case strings.Contains(gitVersion, "+rke2"):
		return "rke2"
	}
	return ""
}

// --- Layer 4: kube-system pod names (legacy fallback) ---
// Useful for local distros (kind, k3d, docker-desktop) that don't set
// providerID or distinctive labels.

var podNameDetectors = []struct {
	substrings  []string
	clusterType string
}{
	{[]string{"-kind-", "kindnet"}, "kind"},
	{[]string{"-minikube"}, "minikube"},
	{[]string{"docker-desktop"}, "docker-desktop"},
	{[]string{"gke-", "-gke-"}, "gke"},
	{[]string{"aws-", "-aws-"}, "eks"},
	{[]string{"azure-", "-azuredisk-", "-azurefile-"}, "aks"},
	{[]string{"openshift", "oc-"}, "openshift"},
	{[]string{"k3d-"}, "k3d"},
	{[]string{"k3s-"}, "k3s"},
	{[]string{"microk8s-"}, "microk8s"},
}

func detectFromKubeSystemPods(ctx context.Context, clientset kubernetes.Interface) string {
	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ""
	}

	for _, pod := range pods.Items {
		for _, d := range podNameDetectors {
			for _, sub := range d.substrings {
				if strings.Contains(pod.Name, sub) {
					return d.clusterType
				}
			}
		}
	}
	return ""
}
