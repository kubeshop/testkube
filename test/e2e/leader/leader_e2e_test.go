package leader_e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/testkube/pkg/coordination/leader"
	leasebackendk8s "github.com/kubeshop/testkube/pkg/repository/leasebackend/k8s"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func kubeConfig(t *testing.T) *rest.Config {
	t.Helper()

	// Prefer in-cluster; fall back to local kubeconfig.
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	if _, err := os.Stat(kubeconfig); err != nil {
		t.Skipf("kubeconfig not found: %v", err)
	}

	cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "failed to build kubeconfig")
	return cfg
}

func TestLeaderCoordinator_K8sLease_Integration(t *testing.T) {
	test.IntegrationTest(t)

	cfg := kubeConfig(t)
	clientset, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ns := fmt.Sprintf("testkube-e2e-%s", utils.RandAlphanum(8))
	_, err = clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create test namespace")
	t.Cleanup(func() {
		_ = clientset.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	})

	backend := leasebackendk8s.NewK8sLeaseBackend(
		clientset,
		"testkube-e2e",
		ns,
		leasebackendk8s.WithLeaseName("testkube-e2e-lease"),
		leasebackendk8s.WithLeaseDuration(2*time.Second),
	)

	var active atomic.Int32
	var c1Starts atomic.Int32
	var c2Starts atomic.Int32

	makeTask := func(id string, counter *atomic.Int32) leader.Task {
		return leader.Task{
			Name: id,
			Start: func(taskCtx context.Context) error {
				counter.Add(1)
				if active.Add(1) > 1 {
					return fmt.Errorf("multiple leaders detected")
				}
				<-taskCtx.Done()
				active.Add(-1)
				return nil
			},
		}
	}

	coord1 := leader.New(backend, "e2e-node-1", "e2e-cluster", nil, leader.WithCheckInterval(300*time.Millisecond))
	coord1.Register(makeTask("leader-1", &c1Starts))

	coord2 := leader.New(backend, "e2e-node-2", "e2e-cluster", nil, leader.WithCheckInterval(300*time.Millisecond))
	coord2.Register(makeTask("leader-2", &c2Starts))

	g, runCtx := errgroup.WithContext(ctx)
	g.Go(func() error { return coord1.Run(runCtx) })
	g.Go(func() error { return coord2.Run(runCtx) })

	// Let coordinators run for a short window.
	time.Sleep(3 * time.Second)
	cancel()

	err = g.Wait()
	require.ErrorIs(t, err, context.Canceled, "coordinators should exit on context cancellation")
	require.Equal(t, int32(0), active.Load(), "all tasks should be stopped")
	require.GreaterOrEqual(t, c1Starts.Load()+c2Starts.Load(), int32(1), "at least one coordinator should acquire leadership")
}
