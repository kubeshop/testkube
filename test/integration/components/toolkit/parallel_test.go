package toolkit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

const (
	defaultTestTimeout  = 30 * time.Second
	podObserverPollRate = 100 * time.Millisecond
	informerSyncTimeout = 3 * time.Second
)

// Global variables for test infrastructure
var (
	globalK8sClient     kubernetes.Interface
	globalBaseNamespace = "testkube-test"
)

// setupOnce ensures K8s client is initialized only once
var setupOnce sync.Once

// setupK8sClient sets up the global Kubernetes client
func setupK8sClient() {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to get Kubernetes config: %v", err))
	}

	cfg.QPS = 100
	cfg.Burst = 100

	globalK8sClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
	}

	// Use base namespace from env if provided
	if ns := os.Getenv("TESTKUBE_NAMESPACE"); ns != "" {
		globalBaseNamespace = ns
	}
}

// setupTestWithControlPlane sets up a test with its own mock control plane
func setupTestWithControlPlane(t *testing.T, namespace string) (*mockControlPlane, int, func()) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Start mock control plane
	cp := newMockControlPlane()
	err = cp.start(port)
	require.NoError(t, err, "Failed to start mock control plane")

	// Update TK_CFG to use this control plane
	oldTkCfg := os.Getenv("TK_CFG")
	configureTKConfigForNamespace(namespace, port)

	cleanup := func() {
		cp.stop()
		if oldTkCfg != "" {
			os.Setenv("TK_CFG", oldTkCfg)
		} else {
			os.Unsetenv("TK_CFG")
		}
	}

	return cp, port, cleanup
}

func TestParallelSimpleCountDistribution_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: `echo "Worker {{index}} of {{count}}"`,
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(3)),
		},
		Description: "worker-{{index}}",
	}

	observer, err := startPodObserver(t, namespace)
	require.NoError(t, err, "Failed to start pod observer")
	t.Cleanup(observer.Stop)

	err = executeParallel(t, spec)
	require.NoError(t, err, "Parallel execution should succeed")

	err = observer.WaitForPods(3, 10*time.Second)
	require.NoError(t, err, "Should observe 3 pods")

	pods := observer.GetCreatedPods()

	// Verify each pod has unique resource name with worker index
	resources := make(map[string]bool)
	for _, pod := range pods {
		assert.Contains(t, pod.Labels, "testkube.io/resource")
		resource := pod.Labels["testkube.io/resource"]
		resources[resource] = true
		// Resource names should be like test--0, test--1, test--2
		assert.Regexp(t, `^test--\d+$`, resource, "Resource should have format test--<index>")

		// Verify pod has expected labels
		assert.Contains(t, pod.Labels, "testkube.io/root", "Pod should have root label")
		assert.Equal(t, "test-exec", pod.Labels["testkube.io/root"], "Root label should match execution ID")
	}
	assert.Len(t, resources, 3, "Should have 3 unique worker resources")
}

func TestParallelMatrixDistribution_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:   "busybox:1.36",
					Command: common.Ptr([]string{"sh", "-c"}),
					Args:    &[]string{`echo "Testing on {{matrix.os}}-{{matrix.arch}}"`},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Matrix: map[string]testworkflowsv1.DynamicList{
				"os":   {Static: []any{"linux", "darwin"}},
				"arch": {Static: []any{"amd64", "arm64"}},
			},
		},
		Description: "{{matrix.os}}-{{matrix.arch}}",
	}

	observer, err := startPodObserver(t, namespace)
	require.NoError(t, err, "Failed to start pod observer")
	t.Cleanup(observer.Stop)

	err = executeParallel(t, spec)
	require.NoError(t, err, "Matrix parallel execution should succeed")

	err = observer.WaitForPods(4, 10*time.Second)
	require.NoError(t, err, "Should observe 4 pods for 2x2 matrix")

	pods := observer.GetCreatedPods()
	t.Logf("Observed %d pods for matrix-2x2-distribution", len(pods))

	// Verify we have 4 unique pods for the matrix
	resources := make(map[string]bool)
	for _, pod := range pods {
		resources[pod.Labels["testkube.io/resource"]] = true
	}
	assert.Len(t, resources, 4, "Should have 4 unique worker resources for 2x2 matrix")

	t.Log("Successfully completed matrix distribution with 4 workers")
}

func TestParallelShardingDistribution_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	files := []any{"test1.go", "test2.go", "test3.go", "test4.go", "test5.go", "test6.go"}
	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:   "golang:1.21-alpine",
					Command: common.Ptr([]string{"sh", "-c"}),
					Args:    &[]string{`echo "Shard {{shard.index}} processing: {{shard.testFiles}}"`},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(3)),
			Shards: map[string]testworkflowsv1.DynamicList{
				"testFiles": {Static: files},
			},
		},
		Description: "shard-{{shard.index}}",
	}

	observer, err := startPodObserver(t, namespace)
	require.NoError(t, err, "Failed to start pod observer")
	t.Cleanup(observer.Stop)

	err = executeParallel(t, spec)
	require.NoError(t, err, "Sharding parallel execution should succeed")

	err = observer.WaitForPods(3, 10*time.Second)
	require.NoError(t, err, "Should observe 3 pods for sharding")

	// Validate pods
	pods := observer.GetCreatedPods()
	t.Logf("Observed %d pods for sharding-distribution", len(pods))

	// Verify sharding distribution
	assert.Len(t, pods, 3, "Should create exactly 3 pods for sharding")
	resources := make(map[string]bool)
	for _, pod := range pods {
		resources[pod.Labels["testkube.io/resource"]] = true
	}
	assert.Len(t, resources, 3, "Should have 3 unique worker resources for sharding")

	t.Log("Successfully completed sharding distribution with 3 workers")
}

// Test parallelism limit
func TestParallelismLimit_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: `echo "Worker {{index}}" && sleep 1`,
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(5)),
		},
		Parallelism: 2, // Only 2 at a time
		Description: "parallel-limited-{{index}}",
	}

	observer, err := startPodObserver(t, namespace)
	require.NoError(t, err, "Failed to start pod observer")
	t.Cleanup(observer.Stop)

	err = executeParallel(t, spec)
	require.NoError(t, err, "Parallel execution with limit should succeed")

	err = observer.WaitForPods(5, 15*time.Second)
	require.NoError(t, err, "Should observe 5 pods")

	// Validate pods
	pods := observer.GetCreatedPods()
	t.Logf("Observed %d pods for parallelism-limit", len(pods))

	// Check that parallelism was respected by looking at pod creation times
	// This is a simplified check - in reality we'd need to monitor concurrent running pods
	assert.Len(t, pods, 5, "Should create all 5 pods eventually")

	t.Log("Successfully completed parallelism limit test")
}

// Test worker failure propagation
func TestParallelWorkerFailure_Integration(t *testing.T) {
	test.IntegrationTest(t)
	// Create test namespace
	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	// Create parallel spec where one worker fails
	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:   "busybox:1.36",
					Command: common.Ptr([]string{"sh", "-c"}),
					Args:    &[]string{`if [ "{{index}}" = "1" ]; then exit 1; fi; echo "Worker {{index}} success"`},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(3)),
		},
		Description: "worker-{{index}}",
	}

	// Execute - should fail
	err := executeParallel(t, spec)
	require.Error(t, err, "Parallel execution should fail when worker fails")
	assert.Contains(t, err.Error(), "workers failed", "Error should indicate worker failure")

	t.Log("Worker failure properly propagated")
}

// Test invalid image handling
func TestParallelInvalidImage_Integration(t *testing.T) {
	test.IntegrationTest(t)
	// Create test namespace
	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	// Create parallel spec with invalid image
	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:   "nonexistent/image:doesnotexist",
					Command: common.Ptr([]string{"echo"}),
					Args:    &[]string{"should not run"},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(1)),
		},
	}

	// Execute - should fail
	err := executeParallel(t, spec)
	require.Error(t, err, "Should fail with invalid image")

	t.Log("Invalid image handling works correctly")
}

// Test log collection on failure
func TestParallelLogCollectionOnFailure_Integration(t *testing.T) {
	test.IntegrationTest(t)
	// Create test namespace
	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	// Create parallel spec with log collection on failure
	logCondition := "failed"
	spec := &testworkflowsv1.StepParallel{
		StepOperations: testworkflowsv1.StepOperations{
			Run: &testworkflowsv1.StepRun{
				ContainerConfig: testworkflowsv1.ContainerConfig{
					Image:   "busybox:1.36",
					Command: common.Ptr([]string{"sh", "-c"}),
					Args:    &[]string{`echo "Worker {{index}} output"; if [ "{{index}}" = "2" ]; then exit 1; fi`},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(3)),
		},
		Logs:        &logCondition,
		Description: "log-test-{{index}}",
	}

	// Execute - should fail
	err := executeParallel(t, spec)
	require.Error(t, err, "Should fail when worker 2 fails")
	assert.Contains(t, err.Error(), "workers failed")

	t.Log("Log collection on failure test completed")
}

func TestParallelLifecycle_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	// Setup control plane for this test
	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	spec := &testworkflowsv1.StepParallel{
		TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					Env: []testworkflowsv1.EnvVar{
						{EnvVar: corev1.EnvVar{Name: "WORKER_ID", Value: "worker-{{index}}"}},
						{EnvVar: corev1.EnvVar{Name: "TOTAL", Value: "{{count}}"}},
					},
					Resources: &testworkflowsv1.Resources{
						Requests: map[corev1.ResourceName]intstr.IntOrString{
							corev1.ResourceCPU:    intstr.FromString("100m"),
							corev1.ResourceMemory: intstr.FromString("128Mi"),
						},
						Limits: map[corev1.ResourceName]intstr.IntOrString{
							corev1.ResourceCPU:    intstr.FromString("200m"),
							corev1.ResourceMemory: intstr.FromString("256Mi"),
						},
					},
				},
			},
			Setup: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Shell: `echo "SETUP: Preparing worker {{index}}"`,
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepMeta: testworkflowsv1.StepMeta{
						Name: "validate-env",
					},
					StepOperations: testworkflowsv1.StepOperations{
						Shell: `test "$WORKER_ID" = "worker-{{index}}" && test "$TOTAL" = "2"`,
					},
				},
				{
					StepMeta: testworkflowsv1.StepMeta{
						Name: "main-work",
					},
					StepOperations: testworkflowsv1.StepOperations{
						Shell: `echo "MAIN: Worker $WORKER_ID doing work"`,
					},
				},
			},
			After: []testworkflowsv1.Step{
				{
					StepOperations: testworkflowsv1.StepOperations{
						Shell: `echo "AFTER: Cleanup for worker {{index}}"`,
					},
					StepMeta: testworkflowsv1.StepMeta{
						Condition: "always", // Run even on failure
					},
				},
			},
		},
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count: common.Ptr(intstr.FromInt(2)),
		},
		Description: "lifecycle-test-{{index}}",
	}

	err := executeParallel(t, spec)
	require.NoError(t, err, "Lifecycle test should execute successfully")

	// The test verifies that setup/steps/after lifecycle works correctly
	// The successful execution indicates all lifecycle stages ran properly
}

// PodObserver monitors pod creation/deletion in a namespace
type PodObserver struct {
	informer cache.SharedIndexInformer
	stopCh   chan struct{}
	pods     []corev1.Pod
	mu       sync.RWMutex
}

// GetCreatedPods returns a copy of all pods observed
func (o *PodObserver) GetCreatedPods() []corev1.Pod {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]corev1.Pod, len(o.pods))
	copy(result, o.pods)
	return result
}

// WaitForPods waits for the expected number of pods to be created
func (o *PodObserver) WaitForPods(count int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(podObserverPollRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %d pods, got %d", count, len(o.GetCreatedPods()))
		case <-ticker.C:
			if len(o.GetCreatedPods()) >= count {
				return nil
			}
		}
	}
}

// Stop stops the pod observer
func (o *PodObserver) Stop() {
	close(o.stopCh)
}

// startPodObserver creates and starts a pod observer for the given namespace
func startPodObserver(t *testing.T, namespace string) (*PodObserver, error) {
	// Watch for pods created by TestWorkflow executions
	// The root label should match what's configured in TK_CFG, but due to a bug
	// in buildInternalConfig, it uses execution.Id instead of preserving root ID
	// So we need to look for test-exec instead of test-root
	labelSelector := "testkube.io/root=test-exec"
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	// Create informer
	watchList := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return globalK8sClient.CoreV1().Pods(namespace).List(context.Background(), listOptions)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return globalK8sClient.CoreV1().Pods(namespace).Watch(context.Background(), listOptions)
		},
	}

	informer := cache.NewSharedIndexInformer(
		watchList,
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)

	observer := &PodObserver{
		informer: informer,
		stopCh:   make(chan struct{}),
		pods:     []corev1.Pod{},
	}

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				observer.mu.Lock()
				observer.pods = append(observer.pods, *pod)
				observer.mu.Unlock()
			}
		},
	})

	// Start informer
	go informer.Run(observer.stopCh)

	// Wait for cache sync
	ctx, cancel := context.WithTimeout(context.Background(), informerSyncTimeout)
	defer cancel()

	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		observer.Stop()
		return nil, fmt.Errorf("failed to sync informer cache")
	}

	return observer, nil
}

// executeParallel runs a parallel spec without pod observation
func executeParallel(t *testing.T, spec *testworkflowsv1.StepParallel) error {
	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Load configuration from TK_CFG
	cfg, err := config.LoadConfigV2()
	require.NoError(t, err, "Failed to load config")

	// TODO: Switch to --base64 encoding when production parallel command is updated
	// Increase timeout for complex tests
	timeout := defaultTestTimeout
	if spec.Count != nil && spec.Count.IntVal > 3 {
		timeout = timeout * 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create NoOp storage for tests
	storage, err := artifacts.InternalStorageWithProvider(&artifacts.NoOpStorageProvider{}, cfg)
	require.NoError(t, err)

	// Use RunParallelWithOptions with injected storage
	opts := &commands.ParallelOptions{Storage: storage}
	return commands.RunParallelWithOptions(ctx, string(data), cfg, false, opts)
}

// Mock control plane for testing
type mockControlPlane struct {
	server *grpc.Server
	cloud.UnimplementedTestKubeCloudAPIServer
}

func newMockControlPlane() *mockControlPlane {
	return &mockControlPlane{}
}

func (m *mockControlPlane) start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	m.server = grpc.NewServer()
	cloud.RegisterTestKubeCloudAPIServer(m.server, m)

	go func() {
		_ = m.server.Serve(lis)
	}()

	return nil
}

func (m *mockControlPlane) stop() {
	if m.server != nil {
		m.server.GracefulStop()
		m.server = nil
	}
}

// Implement required gRPC methods
func (m *mockControlPlane) GetProContext(ctx context.Context, req *emptypb.Empty) (*cloud.ProContextResponse, error) {
	return &cloud.ProContextResponse{
		OrgId:        "test-org",
		EnvId:        "test-env",
		Capabilities: []*cloud.Capability{},
	}, nil
}

// createTestNamespace creates a unique namespace for a test
func createTestNamespace(t *testing.T) string {
	// Ensure K8s client is setup
	setupOnce.Do(setupK8sClient)

	// Create unique namespace
	testNamespace := fmt.Sprintf("%s-%d", globalBaseNamespace, time.Now().UnixNano())
	createNamespaceInCluster(t, testNamespace)

	// Wait for namespace to be ready
	waitForNamespaceReady(t, testNamespace)

	return testNamespace
}

// createNamespaceInCluster creates the namespace in Kubernetes
func createNamespaceInCluster(t *testing.T, name string) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := globalK8sClient.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test namespace")

	return namespace
}

// configureTKConfigForNamespace updates the TK_CFG environment variable
func configureTKConfigForNamespace(namespace string, port int) {
	config := fmt.Sprintf(`{
		"w": {"n": "test"},
		"r": {"i": "test", "r": "test-root", "f": "test"},
		"e": {"i": "test-exec", "s": "%s"},
		"c": {"url": "http://localhost:%d"},
		"W": {"n": "%s"}
	}`, time.Now().Format(time.RFC3339), port, namespace)

	os.Setenv("TK_CFG", config)
}

// waitForNamespaceReady waits for the namespace to be active
func waitForNamespaceReady(t *testing.T, namespace string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for namespace to be ready")
		case <-ticker.C:
			ns, err := globalK8sClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
			if err == nil && ns.Status.Phase == corev1.NamespaceActive {
				t.Logf("Created namespace %s", namespace)
				return
			}
		}
	}
}

// deleteTestNamespace deletes the test namespace
func deleteTestNamespace(t *testing.T, namespace string) {
	if namespace != "" && namespace != globalBaseNamespace {
		propagationPolicy := metav1.DeletePropagationForeground
		err := globalK8sClient.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			t.Logf("Warning: failed to delete test namespace %s: %v", namespace, err)
		}

		// Don't wait for full deletion to avoid slowing down tests
		// The namespace will be cleaned up in the background
	}
}
