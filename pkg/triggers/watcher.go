package triggers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsinformerv1 "k8s.io/client-go/informers/apps/v1"
	coreinformerv1 "k8s.io/client-go/informers/core/v1"
	networkinginformerv1 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	testkubeinformerv1 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/tests/v1"
	testkubeinformerv2 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/tests/v2"
	testkubeinformerv3 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/tests/v3"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	testsuitev2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/informers/externalversions"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
)

type k8sInformers struct {
	podInformers          []coreinformerv1.PodInformer
	deploymentInformers   []appsinformerv1.DeploymentInformer
	daemonsetInformers    []appsinformerv1.DaemonSetInformer
	statefulsetInformers  []appsinformerv1.StatefulSetInformer
	serviceInformers      []coreinformerv1.ServiceInformer
	ingressInformers      []networkinginformerv1.IngressInformer
	clusterEventInformers []coreinformerv1.EventInformer
	configMapInformers    []coreinformerv1.ConfigMapInformer

	testTriggerInformers []testkubeinformerv1.TestTriggerInformer
	testSuiteInformers   []testkubeinformerv2.TestSuiteInformer
	testInformers        []testkubeinformerv3.TestInformer
}

func newK8sInformers(clientset kubernetes.Interface, testKubeClientset versioned.Interface, testkubeNamespace string, watcherNamespaces []string, watchTestkubeCrAllNamespaces bool) *k8sInformers {
	var k8sInformers k8sInformers
	if len(watcherNamespaces) == 0 {
		watcherNamespaces = append(watcherNamespaces, v1.NamespaceAll)
	}

	for _, namespace := range watcherNamespaces {
		f := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace(namespace))
		k8sInformers.podInformers = append(k8sInformers.podInformers, f.Core().V1().Pods())
		k8sInformers.deploymentInformers = append(k8sInformers.deploymentInformers, f.Apps().V1().Deployments())
		k8sInformers.daemonsetInformers = append(k8sInformers.daemonsetInformers, f.Apps().V1().DaemonSets())
		k8sInformers.statefulsetInformers = append(k8sInformers.statefulsetInformers, f.Apps().V1().StatefulSets())
		k8sInformers.serviceInformers = append(k8sInformers.serviceInformers, f.Core().V1().Services())
		k8sInformers.ingressInformers = append(k8sInformers.ingressInformers, f.Networking().V1().Ingresses())
		k8sInformers.clusterEventInformers = append(k8sInformers.clusterEventInformers, f.Core().V1().Events())
		k8sInformers.configMapInformers = append(k8sInformers.configMapInformers, f.Core().V1().ConfigMaps())

	}

	var watchedNamespaces string
	if watchTestkubeCrAllNamespaces == true {
		watchedNamespaces = v1.NamespaceAll
	} else {
		watchedNamespaces = testkubeNamespace
	}
	var testkubeInformerFactory externalversions.SharedInformerFactory
	testkubeInformerFactory = externalversions.NewSharedInformerFactoryWithOptions(
		testKubeClientset, 0, externalversions.WithNamespace(watchedNamespaces))
	k8sInformers.testTriggerInformers = append(k8sInformers.testTriggerInformers, testkubeInformerFactory.Tests().V1().TestTriggers())
	k8sInformers.testSuiteInformers = append(k8sInformers.testSuiteInformers, testkubeInformerFactory.Tests().V2().TestSuites())
	k8sInformers.testInformers = append(k8sInformers.testInformers, testkubeInformerFactory.Tests().V3().Tests())

	return &k8sInformers
}

func (s *Service) runWatcher(ctx context.Context, leaseChan chan bool) {
	running := false
	var stopChan chan struct{}

	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("trigger service: stopping watcher component: context finished")
			if _, ok := <-stopChan; ok {
				close(stopChan)
			}
			return
		case leased := <-leaseChan:
			if !leased {
				if running {
					s.logger.Infof("trigger service: instance %s in cluster %s lost lease", s.identifier, s.clusterID)
					close(stopChan)
					s.informers = nil
					running = false
				}
			} else {
				if !running {
					s.logger.Infof("trigger service: instance %s in cluster %s acquired lease", s.identifier, s.clusterID)
					s.informers = newK8sInformers(s.clientset, s.testKubeClientset, s.testkubeNamespace, s.watcherNamespaces, s.watchTestkubeCrAllNamespaces)
					stopChan = make(chan struct{})
					s.runInformers(ctx, stopChan)
					running = true
				}
			}
		}
	}
}

func (s *Service) runInformers(ctx context.Context, stop <-chan struct{}) {
	if s.informers == nil {
		s.logger.Errorf("trigger service: error running k8s informers: informers are nil")
		return
	}

	for i := range s.informers.podInformers {
		s.informers.podInformers[i].Informer().AddEventHandler(s.podEventHandler(ctx))
	}

	for i := range s.informers.deploymentInformers {
		s.informers.deploymentInformers[i].Informer().AddEventHandler(s.deploymentEventHandler(ctx))
	}

	for i := range s.informers.daemonsetInformers {
		s.informers.daemonsetInformers[i].Informer().AddEventHandler(s.daemonSetEventHandler(ctx))
	}

	for i := range s.informers.statefulsetInformers {
		s.informers.statefulsetInformers[i].Informer().AddEventHandler(s.statefulSetEventHandler(ctx))
	}

	for i := range s.informers.serviceInformers {
		s.informers.serviceInformers[i].Informer().AddEventHandler(s.serviceEventHandler(ctx))
	}

	for i := range s.informers.ingressInformers {
		s.informers.ingressInformers[i].Informer().AddEventHandler(s.ingressEventHandler(ctx))
	}

	for i := range s.informers.clusterEventInformers {
		s.informers.clusterEventInformers[i].Informer().AddEventHandler(s.clusterEventEventHandler(ctx))
	}

	for i := range s.informers.configMapInformers {
		s.informers.configMapInformers[i].Informer().AddEventHandler(s.configMapEventHandler(ctx))
	}

	for i := range s.informers.testTriggerInformers {
		s.informers.testTriggerInformers[i].Informer().AddEventHandler(s.testTriggerEventHandler())
	}
	for i := range s.informers.testSuiteInformers {
		s.informers.testSuiteInformers[i].Informer().AddEventHandler(s.testSuiteEventHandler())
	}
	for i := range s.informers.testInformers {
		s.informers.testInformers[i].Informer().AddEventHandler(s.testEventHandler())
	}

	s.logger.Debugf("trigger service: starting pod informers")
	for i := range s.informers.podInformers {
		go s.informers.podInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting deployment informers")
	for i := range s.informers.deploymentInformers {
		go s.informers.deploymentInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting daemonset informers")
	for i := range s.informers.daemonsetInformers {
		go s.informers.daemonsetInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting statefulset informers")
	for i := range s.informers.statefulsetInformers {
		go s.informers.statefulsetInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting service informers")
	for i := range s.informers.serviceInformers {
		go s.informers.serviceInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting ingress informers")
	for i := range s.informers.ingressInformers {
		go s.informers.ingressInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting cluster event informers")
	for i := range s.informers.clusterEventInformers {
		go s.informers.clusterEventInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting config map informers")
	for i := range s.informers.configMapInformers {
		go s.informers.configMapInformers[i].Informer().Run(stop)
	}

	s.logger.Debugf("trigger service: starting test trigger informer")
	for i := range s.informers.testTriggerInformers {
		go s.informers.testTriggerInformers[i].Informer().Run(stop)
	}
	s.logger.Debugf("trigger service: starting test suite informer")
	for i := range s.informers.testSuiteInformers {
		go s.informers.testSuiteInformers[i].Informer().Run(stop)
	}
	s.logger.Debugf("trigger service: starting test informer")
	for i := range s.informers.testInformers {
		go s.informers.testInformers[i].Informer().Run(stop)
	}
}

func (s *Service) podEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getPodConditions(ctx, s.clientset, object)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(pod.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: pod %s/%s was created in the past",
					pod.Namespace, pod.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s created", pod.Namespace, pod.Name)
			event := newWatcherEvent(testtrigger.EventCreated, pod, testtrigger.ResourcePod, withConditionsGetter(getConditions(pod)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create pod event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process delete pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s deleted", pod.Namespace, pod.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, pod, testtrigger.ResourcePod, withConditionsGetter(getConditions(pod)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete pod event: %v", err)
			}
		},
	}
}

func (s *Service) deploymentEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getDeploymentConditions(ctx, s.clientset, object)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(deployment.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: deployment %s/%s was created in the past",
					deployment.Namespace, deployment.Name,
				)
				return
			}
			s.logger.Debugf("emiting event: deployment %s/%s created", deployment.Namespace, deployment.Name)
			event := newWatcherEvent(testtrigger.EventCreated, deployment, testtrigger.ResourceDeployment, withConditionsGetter(getConditions(deployment)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create deployment event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok := oldObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for old object due to it being an unexpected type, received type %+v",
					oldDeployment,
				)
				return
			}
			newDeployment, ok := newObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for new object due to it being an unexpected type, received type %+v",
					newDeployment,
				)
				return
			}
			if cmp.Equal(oldDeployment.Spec, newDeployment.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: deployment specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: deployment %s/%s updated",
				newDeployment.Namespace, newDeployment.Name,
			)
			causes := diffDeployments(oldDeployment, newDeployment)
			event := newWatcherEvent(testtrigger.EventModified, newDeployment, testtrigger.ResourceDeployment, withCauses(causes), withConditionsGetter(getConditions(newDeployment)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update deployment event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: deployment %s/%s deleted", deployment.Namespace, deployment.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, deployment, testtrigger.ResourceDeployment, withConditionsGetter(getConditions(deployment)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete deployment event: %v", err)
			}
		},
	}
}

func (s *Service) statefulSetEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getStatefulSetConditions(ctx, s.clientset, object)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			statefulset, ok := obj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf("failed to process create statefulset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(statefulset.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: statefulset %s/%s was created in the past",
					statefulset.Namespace, statefulset.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: statefulset %s/%s created", statefulset.Namespace, statefulset.Name)
			event := newWatcherEvent(testtrigger.EventCreated, statefulset, testtrigger.ResourceStatefulSet, withConditionsGetter(getConditions(statefulset)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create statefulset event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldStatefulSet, ok := oldObj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update statefulset event for old object due to it being an unexpected type, received type %+v",
					oldStatefulSet,
				)
				return
			}
			newStatefulSet, ok := newObj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update statefulset event for new object due to it being an unexpected type, received type %+v",
					newStatefulSet,
				)
				return
			}
			if cmp.Equal(oldStatefulSet.Spec, newStatefulSet.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: statefulset specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: statefulset %s/%s updated",
				newStatefulSet.Namespace, newStatefulSet.Name,
			)
			event := newWatcherEvent(testtrigger.EventModified, newStatefulSet, testtrigger.ResourceStatefulSet, withConditionsGetter(getConditions(newStatefulSet)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update statefulset event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			statefulset, ok := obj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf("failed to process delete statefulset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: statefulset %s/%s deleted", statefulset.Namespace, statefulset.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, statefulset, testtrigger.ResourceStatefulSet, withConditionsGetter(getConditions(statefulset)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete statefulset event: %v", err)
			}
		},
	}
}

func (s *Service) daemonSetEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getDaemonSetConditions(ctx, s.clientset, object)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			daemonset, ok := obj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf("failed to process create daemonset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(daemonset.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: daemonset %s/%s was created in the past",
					daemonset.Namespace, daemonset.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: daemonset %s/%s created", daemonset.Namespace, daemonset.Name)
			event := newWatcherEvent(testtrigger.EventCreated, daemonset, testtrigger.ResourceDaemonSet, withConditionsGetter(getConditions(daemonset)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create daemonset event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDaemonSet, ok := oldObj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update daemonset event for old object due to it being an unexpected type, received type %+v",
					oldDaemonSet,
				)
				return
			}
			newDaemonSet, ok := newObj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update daemonset event for new object due to it being an unexpected type, received type %+v",
					newDaemonSet,
				)
				return
			}
			if cmp.Equal(oldDaemonSet.Spec, newDaemonSet.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: daemonset specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: daemonset %s/%s updated",
				newDaemonSet.Namespace, newDaemonSet.Name,
			)
			event := newWatcherEvent(testtrigger.EventModified, newDaemonSet, testtrigger.ResourceDaemonSet, withConditionsGetter(getConditions(newDaemonSet)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update daemonset event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			daemonset, ok := obj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf("failed to process delete daemonset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: daemonset %s/%s deleted", daemonset.Namespace, daemonset.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, daemonset, testtrigger.ResourceDaemonSet, withConditionsGetter(getConditions(daemonset)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete daemonset event: %v", err)
			}
		},
	}
}

func (s *Service) serviceEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getServiceConditions(ctx, s.clientset, object)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			service, ok := obj.(*corev1.Service)
			if !ok {
				s.logger.Errorf("failed to process create service event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(service.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: service %s/%s was created in the past",
					service.Namespace, service.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: service %s/%s created", service.Namespace, service.Name)
			event := newWatcherEvent(testtrigger.EventCreated, service, testtrigger.ResourceService, withConditionsGetter(getConditions(service)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create service event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldService, ok := oldObj.(*corev1.Service)
			if !ok {
				s.logger.Errorf(
					"failed to process update service event for old object due to it being an unexpected type, received type %+v",
					oldService,
				)
				return
			}
			newService, ok := newObj.(*corev1.Service)
			if !ok {
				s.logger.Errorf(
					"failed to process update service event for new object due to it being an unexpected type, received type %+v",
					newService,
				)
				return
			}
			if cmp.Equal(oldService.Spec, newService.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: service specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: service %s/%s updated",
				newService.Namespace, newService.Name,
			)
			event := newWatcherEvent(testtrigger.EventModified, newService, testtrigger.ResourceService, withConditionsGetter(getConditions(newService)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update service event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			service, ok := obj.(*corev1.Service)
			if !ok {
				s.logger.Errorf("failed to process delete service event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: service %s/%s deleted", service.Namespace, service.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, service, testtrigger.ResourceService, withConditionsGetter(getConditions(service)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete service event: %v", err)
			}
		},
	}
}

func (s *Service) ingressEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ingress, ok := obj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf("failed to process create ingress event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(ingress.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: ingress %s/%s was created in the past",
					ingress.Namespace, ingress.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: ingress %s/%s created", ingress.Namespace, ingress.Name)
			event := newWatcherEvent(testtrigger.EventCreated, ingress, testtrigger.ResourceIngress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create ingress event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldIngress, ok := oldObj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf(
					"failed to process update ingress event for old object due to it being an unexpected type, received type %+v",
					oldIngress,
				)
				return
			}
			newIngress, ok := newObj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf(
					"failed to process update ingress event for new object due to it being an unexpected type, received type %+v",
					newIngress,
				)
				return
			}
			if cmp.Equal(oldIngress.Spec, newIngress.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: ingress specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: ingress %s/%s updated",
				oldIngress.Namespace, newIngress.Name,
			)
			event := newWatcherEvent(testtrigger.EventModified, newIngress, testtrigger.ResourceIngress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update ingress event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			ingress, ok := obj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf("failed to process delete ingress event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: ingress %s/%s deleted", ingress.Namespace, ingress.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, ingress, testtrigger.ResourceIngress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete ingress event: %v", err)
			}
		},
	}
}

func (s *Service) clusterEventEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			clusterEvent, ok := obj.(*corev1.Event)
			if !ok {
				s.logger.Errorf("failed to process create cluster event event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(clusterEvent.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: cluster event %s/%s was created in the past",
					clusterEvent.Namespace, clusterEvent.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: cluster event %s/%s created", clusterEvent.Namespace, clusterEvent.Name)
			event := newWatcherEvent(testtrigger.EventCreated, clusterEvent, testtrigger.ResourceEvent)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create cluster event event: %v", err)
			}
		},
	}
}

func (s *Service) testTriggerEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggersv1.TestTrigger)
			if !ok {
				s.logger.Errorf("failed to process create testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.addTrigger(t)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			t, ok := newObj.(*testtriggersv1.TestTrigger)
			if !ok {
				s.logger.Errorf(
					"failed to process update testtrigger event for new testtrigger due to it being an unexpected type, received type %+v",
					newObj,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: updating testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.updateTrigger(t)
		},
		DeleteFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggersv1.TestTrigger)
			if !ok {
				s.logger.Errorf("failed to process delete testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: deleting testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.removeTrigger(t)
		},
	}
}

func (s *Service) testSuiteEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			testSuite, ok := obj.(*testsuitev2.TestSuite)
			if !ok {
				s.logger.Errorf("failed to process create testsuite event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(testSuite.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: testsuite %s/%s was created in the past",
					testSuite.Namespace, testSuite.Name,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding testsuite %s/%s",
				testSuite.Namespace, testSuite.Name,
			)
			s.addTestSuite(testSuite)
		},
	}
}

func (s *Service) testEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			test, ok := obj.(*testsv3.Test)
			if !ok {
				s.logger.Errorf("failed to process create test event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(test.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: test %s/%s was created in the past",
					test.Namespace, test.Name,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding test %s/%s",
				test.Namespace, test.Name,
			)
			s.addTest(test)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			test, ok := newObj.(*testsv3.Test)
			if !ok {
				s.logger.Errorf("failed to process update test event due to it being an unexpected type, received type %+v", newObj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: updating test %s/%s",
				test.Namespace, test.Name,
			)
			s.updateTest(test)
		},
	}
}

func (s *Service) configMapEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			configMap, ok := obj.(*corev1.ConfigMap)
			if !ok {
				s.logger.Errorf("failed to process create config map event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(configMap.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: config map %s/%s was created in the past",
					configMap.Namespace, configMap.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: config map %s/%s created", configMap.Namespace, configMap.Name)
			event := newWatcherEvent(testtrigger.EventCreated, configMap, testtrigger.ResourceConfigMap)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create config map event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldConfigMap, ok := oldObj.(*corev1.ConfigMap)
			if !ok {
				s.logger.Errorf(
					"failed to process update config map event for old object due to it being an unexpected type, received type %+v",
					oldConfigMap,
				)
				return
			}
			newConfigMap, ok := newObj.(*corev1.ConfigMap)
			if !ok {
				s.logger.Errorf(
					"failed to process update config map event for new object due to it being an unexpected type, received type %+v",
					newConfigMap,
				)
				return
			}
			if cmp.Equal(oldConfigMap.Data, newConfigMap.Data) && cmp.Equal(oldConfigMap.BinaryData, newConfigMap.BinaryData) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: config map data and binary data are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: config map %s/%s updated",
				oldConfigMap.Namespace, newConfigMap.Name,
			)
			event := newWatcherEvent(testtrigger.EventModified, newConfigMap, testtrigger.ResourceConfigMap)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update config map event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			configMap, ok := obj.(*corev1.ConfigMap)
			if !ok {
				s.logger.Errorf("failed to process delete config map event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: config map %s/%s deleted", configMap.Namespace, configMap.Name)
			event := newWatcherEvent(testtrigger.EventDeleted, configMap, testtrigger.ResourceConfigMap)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete config map event: %v", err)
			}
		},
	}
}
