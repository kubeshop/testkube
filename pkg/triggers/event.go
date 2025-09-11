package triggers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/mapper/daemonsets"
	"github.com/kubeshop/testkube/pkg/mapper/deployments"
	"github.com/kubeshop/testkube/pkg/mapper/k8sevents"
	"github.com/kubeshop/testkube/pkg/mapper/pods"
	"github.com/kubeshop/testkube/pkg/mapper/services"
	"github.com/kubeshop/testkube/pkg/mapper/statefulsets"
)

const testkubeEventCausePrefix = "event-"

type conditionsGetterFn func() ([]testtriggersv1.TestTriggerCondition, error)

type addressGetterFn func(ctx context.Context, delay time.Duration) (string, error)

type watcherEvent struct {
	resource         testtrigger.ResourceType
	name             string
	Namespace        string `json:"namespace"`
	labels           map[string]string
	objectMeta       metav1.Object
	Object           any `json:"object"`
	eventType        testtrigger.EventType
	causes           []testtrigger.Cause
	conditionsGetter conditionsGetterFn
	addressGetter    addressGetterFn
	Agent            watcherAgent `json:"agent"`
}

// watcherAgent represents agent context exposed to templates and JSONPath
type watcherAgent struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Data   map[string]string `json:"data,omitempty"`
}

// loadDataFromFile loads key-value pairs from a file into a map[string]string.
// The file format supports:
// - key=value pairs (one per line)
// - Empty lines are ignored
// - Leading and trailing whitespace is trimmed
func loadDataFromFile(filePath string) (map[string]string, error) {
	if filePath == "" {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty map without error
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to open agent data file %s: %w", filePath, err)
	}
	defer file.Close()

	data := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d in %s: expected key=value, got: %s", lineNum, filePath, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty key at line %d in %s", lineNum, filePath)
		}

		data[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading agent data file %s: %w", filePath, err)
	}

	return data, nil
}

type watcherOpts func(*watcherEvent)

func withCauses(causes []testtrigger.Cause) watcherOpts {
	return func(w *watcherEvent) {
		w.causes = causes
	}
}

func withConditionsGetter(conditionsGetter conditionsGetterFn) watcherOpts {
	return func(w *watcherEvent) {
		w.conditionsGetter = conditionsGetter
	}
}

func withAddressGetter(addressGetter addressGetterFn) watcherOpts {
	return func(w *watcherEvent) {
		w.addressGetter = addressGetter
	}
}

func withNotEmptyName(name string) watcherOpts {
	return func(w *watcherEvent) {
		if name != "" {
			w.name = name
		}
	}
}

func newWatcherEvent(
	eventType testtrigger.EventType,
	objectMeta metav1.Object,
	object any,
	resource testtrigger.ResourceType,
	agent watcherAgent,
	opts ...watcherOpts,
) *watcherEvent {
	w := &watcherEvent{
		resource:   resource,
		name:       objectMeta.GetName(),
		Namespace:  objectMeta.GetNamespace(),
		labels:     objectMeta.GetLabels(),
		objectMeta: objectMeta,
		Object:     object,
		eventType:  eventType,
		Agent:      agent,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

func getPodConditions(ctx context.Context, clientset kubernetes.Interface, object metav1.Object) ([]testtriggersv1.TestTriggerCondition, error) {
	pod, err := clientset.CoreV1().Pods(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pods.MapCRDConditionsToAPI(pod.Status.Conditions, time.Now()), nil
}

func getPodAdress(ctx context.Context, clientset kubernetes.Interface, object metav1.Object, delay time.Duration) (string, error) {
	podIP := ""
outerLoop:
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			pod, err := clientset.CoreV1().Pods(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
			if err != nil {
				return "", err
			}

			podIP = pod.Status.PodIP
			if podIP != "" {
				break outerLoop
			}

			time.Sleep(delay)
		}
	}

	return fmt.Sprintf("%s.%s.pod.cluster.local", strings.ReplaceAll(podIP, ".", "-"), object.GetNamespace()), nil
}

func getDeploymentConditions(
	ctx context.Context,
	clientset kubernetes.Interface,
	object metav1.Object,
) ([]testtriggersv1.TestTriggerCondition, error) {
	deployment, err := clientset.AppsV1().Deployments(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return deployments.MapCRDConditionsToAPI(deployment.Status.Conditions, time.Now()), nil
}

func getDaemonSetConditions(
	ctx context.Context,
	clientset kubernetes.Interface,
	object metav1.Object,
) ([]testtriggersv1.TestTriggerCondition, error) {
	daemonset, err := clientset.AppsV1().DaemonSets(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return daemonsets.MapCRDConditionsToAPI(daemonset.Status.Conditions, time.Now()), nil
}

func getStatefulSetConditions(
	ctx context.Context,
	clientset kubernetes.Interface,
	object metav1.Object,
) ([]testtriggersv1.TestTriggerCondition, error) {
	statefulset, err := clientset.AppsV1().StatefulSets(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return statefulsets.MapCRDConditionsToAPI(statefulset.Status.Conditions, time.Now()), nil
}

func getServiceConditions(
	ctx context.Context,
	clientset kubernetes.Interface,
	object metav1.Object,
) ([]testtriggersv1.TestTriggerCondition, error) {
	service, err := clientset.CoreV1().Services(object.GetNamespace()).Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return services.MapCRDConditionsToAPI(service.Status.Conditions, time.Now()), nil
}

func getServiceAdress(ctx context.Context, clientset kubernetes.Interface, object metav1.Object) (string, error) {
	return fmt.Sprintf("%s.%s.svc.cluster.local", object.GetName(), object.GetNamespace()), nil
}

func getTestkubeEventNameAndCauses(event *corev1.Event) (string, []testtrigger.Cause) {
	var causes []testtrigger.Cause
	if !strings.HasPrefix(event.Name, k8sevents.TestkubeEventPrefix) {
		return "", causes
	}

	causes = append(causes, testtrigger.Cause(fmt.Sprintf("%s%s", testkubeEventCausePrefix, event.Reason)))
	return event.InvolvedObject.Name, causes
}
