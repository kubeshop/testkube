package triggers

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/mapper/daemonsets"
	"github.com/kubeshop/testkube/pkg/mapper/deployments"
	"github.com/kubeshop/testkube/pkg/mapper/k8sevents"
	"github.com/kubeshop/testkube/pkg/mapper/pods"
	"github.com/kubeshop/testkube/pkg/mapper/services"
	"github.com/kubeshop/testkube/pkg/mapper/statefulsets"
	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

const testkubeEventCausePrefix = "event-"

type conditionsGetterFn func() ([]testtriggersv1.TestTriggerCondition, error)

type addressGetterFn func(ctx context.Context, delay time.Duration) (string, error)

type watcherEvent struct {
	name             string
	Namespace        string `json:"namespace"`
	resource         testtrigger.ResourceType
	resourceLabels   map[string]string
	objectMeta       metav1.Object
	Object           any `json:"object"`
	eventType        testtrigger.EventType
	causes           []testtrigger.Cause
	conditionsGetter conditionsGetterFn
	addressGetter    addressGetterFn
	EventLabels      map[string]string `json:"eventLabels"`
	Agent            watcherAgent      `json:"agent"`
}

// watcherAgent represents agent context exposed to templates and JSONPath
type watcherAgent struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
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

const (
	eventLabelKeyAgentName         string = "testkube.io/agent-name"
	eventLabelKeyAgentNamespace    string = "testkube.io/agent-namespace"
	eventLabelKeyResourceName      string = "testkube.io/resource-name"
	eventLabelKeyResourceNamespace string = "testkube.io/resource-namespace"
	eventLabelKeyResourceKind      string = "testkube.io/resource-kind"
	eventLabelKeyResourceGroup     string = "testkube.io/resource-group"
	eventLabelKeyResourceVersion   string = "testkube.io/resource-version"
)

func (s Service) newWatcherEvent(
	eventType testtrigger.EventType,
	objectMeta metav1.Object,
	object any,
	resource testtrigger.ResourceType,
	opts ...watcherOpts,
) *watcherEvent {
	w := &watcherEvent{
		resource:       resource,
		name:           objectMeta.GetName(),
		Namespace:      objectMeta.GetNamespace(),
		resourceLabels: objectMeta.GetLabels(),
		objectMeta:     objectMeta,
		Object:         object,
		eventType:      eventType,
		EventLabels:    map[string]string{},
		Agent:          s.Agent,
	}

	maps.Copy(w.EventLabels, s.eventLabels)
	w.EventLabels[eventLabelKeyAgentName] = s.agentName
	w.EventLabels[eventLabelKeyAgentNamespace] = s.testkubeNamespace
	w.EventLabels[eventLabelKeyResourceName] = objectMeta.GetName()
	w.EventLabels[eventLabelKeyResourceNamespace] = objectMeta.GetNamespace()

	if runtimeObject, ok := object.(runtime.Object); ok &&
		s.informers != nil && s.informers.scheme != nil {
		gvks, _, err := s.informers.scheme.ObjectKinds(runtimeObject)
		if err != nil {
			s.logger.Warnf("error getting object kinds from scheme, skipped adding event label: %v", err)
		} else if len(gvks) > 0 {
			gvk := gvks[0]
			w.EventLabels[eventLabelKeyResourceKind] = gvk.Kind
			w.EventLabels[eventLabelKeyResourceGroup] = gvk.Group
			w.EventLabels[eventLabelKeyResourceVersion] = gvk.Version
		}
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
