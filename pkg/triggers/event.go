package triggers

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/mapper/daemonsets"
	"github.com/kubeshop/testkube/pkg/mapper/deployments"
	"github.com/kubeshop/testkube/pkg/mapper/pods"
	"github.com/kubeshop/testkube/pkg/mapper/services"
	"github.com/kubeshop/testkube/pkg/mapper/statefulsets"
)

type conditionsGetterFn func() ([]testtriggersv1.TestTriggerCondition, error)

type addressGetterFn func(ctx context.Context, delay time.Duration) (string, error)

type watcherEvent struct {
	resource         testtrigger.ResourceType
	name             string
	namespace        string
	labels           map[string]string
	object           metav1.Object
	eventType        testtrigger.EventType
	causes           []testtrigger.Cause
	conditionsGetter conditionsGetterFn
	addressGetter    addressGetterFn
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

func newWatcherEvent(
	eventType testtrigger.EventType,
	object metav1.Object,
	resource testtrigger.ResourceType,
	opts ...watcherOpts,
) *watcherEvent {
	w := &watcherEvent{
		resource:  resource,
		name:      object.GetName(),
		namespace: object.GetNamespace(),
		labels:    object.GetLabels(),
		object:    object,
		eventType: eventType,
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
