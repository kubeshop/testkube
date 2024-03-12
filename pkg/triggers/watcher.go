package triggers

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	appsinformerv1 "k8s.io/client-go/informers/apps/v1"
	coreinformerv1 "k8s.io/client-go/informers/core/v1"
	networkinginformerv1 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	testsourcev1 "github.com/kubeshop/testkube-operator/api/testsource/v1"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"

	testsuitev3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube-operator/pkg/informers/externalversions"
	testkubeexecutorinformerv1 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/executor/v1"
	testkubeinformerv1 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/tests/v1"

	testkubeinformerv3 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/tests/v3"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
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

	testTriggerInformer testkubeinformerv1.TestTriggerInformer
	testSuiteInformer   testkubeinformerv3.TestSuiteInformer
	testInformer        testkubeinformerv3.TestInformer
	executorInformer    testkubeexecutorinformerv1.ExecutorInformer
	webhookInformer     testkubeexecutorinformerv1.WebhookInformer
	testSourceInformer  testkubeinformerv1.TestSourceInformer
}

func newK8sInformers(clientset kubernetes.Interface, testKubeClientset versioned.Interface,
	testkubeNamespace string, watcherNamespaces []string) *k8sInformers {
	var k8sInformers k8sInformers
	if len(watcherNamespaces) == 0 {
		watcherNamespaces = append(watcherNamespaces, metav1.NamespaceAll)
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

	testkubeInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(
		testKubeClientset, 0, externalversions.WithNamespace(testkubeNamespace))
	k8sInformers.testTriggerInformer = testkubeInformerFactory.Tests().V1().TestTriggers()
	k8sInformers.testSuiteInformer = testkubeInformerFactory.Tests().V3().TestSuites()
	k8sInformers.testInformer = testkubeInformerFactory.Tests().V3().Tests()
	k8sInformers.executorInformer = testkubeInformerFactory.Executor().V1().Executor()
	k8sInformers.webhookInformer = testkubeInformerFactory.Executor().V1().Webhook()
	k8sInformers.testSourceInformer = testkubeInformerFactory.Tests().V1().TestSource()

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
					s.informers = newK8sInformers(s.clientset, s.testKubeClientset, s.testkubeNamespace, s.watcherNamespaces)
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

	s.informers.testTriggerInformer.Informer().AddEventHandler(s.testTriggerEventHandler())
	s.informers.testSuiteInformer.Informer().AddEventHandler(s.testSuiteEventHandler())
	s.informers.testInformer.Informer().AddEventHandler(s.testEventHandler())
	s.informers.executorInformer.Informer().AddEventHandler(s.executorEventHandler())
	s.informers.webhookInformer.Informer().AddEventHandler(s.webhookEventHandler())
	s.informers.testSourceInformer.Informer().AddEventHandler(s.testSourceEventHandler())

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
	go s.informers.testTriggerInformer.Informer().Run(stop)
	s.logger.Debugf("trigger service: starting test suite informer")
	go s.informers.testSuiteInformer.Informer().Run(stop)
	s.logger.Debugf("trigger service: starting test informer")
	go s.informers.testInformer.Informer().Run(stop)
	s.logger.Debugf("trigger service: starting executor informer")
	go s.informers.executorInformer.Informer().Run(stop)
	s.logger.Debugf("trigger service: starting webhook informer")
	go s.informers.webhookInformer.Informer().Run(stop)
	s.logger.Debugf("trigger service: starting test source informer")
	go s.informers.testSourceInformer.Informer().Run(stop)
}

func (s *Service) podEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	getConditions := func(object metav1.Object) func() ([]testtriggersv1.TestTriggerCondition, error) {
		return func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getPodConditions(ctx, s.clientset, object)
		}
	}
	getAddrress := func(object metav1.Object) func(c context.Context, delay time.Duration) (string, error) {
		return func(c context.Context, delay time.Duration) (string, error) {
			return getPodAdress(c, s.clientset, object, delay)
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
			event := newWatcherEvent(testtrigger.EventCreated, pod, testtrigger.ResourcePod,
				withConditionsGetter(getConditions(pod)), withAddressGetter(getAddrress(pod)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create pod event: %v", err)
			}

		},
		UpdateFunc: func(oldObj, newObj any) {
			oldPod, ok := oldObj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process update pod event due to it being an unexpected type, received type %+v", oldObj)
				return
			}
			if inPast(oldPod.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op update trigger: pod %s/%s was updated in the past",
					oldPod.Namespace, oldPod.Name,
				)
				return
			}

			newPod, ok := newObj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process update pod event due to it being an unexpected type, received type %+v", newObj)
				return
			}
			if inPast(newPod.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op update trigger: pod %s/%s was updated in the past",
					newPod.Namespace, newPod.Name,
				)
				return
			}
			if oldPod.Namespace == s.testkubeNamespace && oldPod.Labels["job-name"] != "" && oldPod.Labels[testkube.TestLabelTestName] != "" &&
				newPod.Namespace == s.testkubeNamespace && newPod.Labels["job-name"] != "" && newPod.Labels[testkube.TestLabelTestName] != "" &&
				oldPod.Labels["job-name"] == newPod.Labels["job-name"] {
				s.checkExecutionPodStatus(ctx, oldPod.Labels["job-name"], []*corev1.Pod{oldPod, newPod})
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process delete pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s deleted", pod.Namespace, pod.Name)
			if pod.Namespace == s.testkubeNamespace && pod.Labels["job-name"] != "" && pod.Labels[testkube.TestLabelTestName] != "" {
				s.checkExecutionPodStatus(ctx, pod.Labels["job-name"], []*corev1.Pod{pod})
			}
			event := newWatcherEvent(testtrigger.EventDeleted, pod, testtrigger.ResourcePod,
				withConditionsGetter(getConditions(pod)), withAddressGetter(getAddrress(pod)))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete pod event: %v", err)
			}
		},
	}
}

func (s *Service) checkExecutionPodStatus(ctx context.Context, executionID string, pods []*corev1.Pod) error {
	if len(pods) > 0 && pods[0].Labels[testworkflowprocessor.ExecutionIdLabelName] != "" {
		return nil
	}

	execution, err := s.resultRepository.Get(ctx, executionID)
	if err != nil {
		s.logger.Errorf("get execution returned an error %v while looking for execution id: %s", err, executionID)
		return err
	}

	if execution.ExecutionResult.IsRunning() || execution.ExecutionResult.IsQueued() {
		errorMessage := ""
		for _, pod := range pods {
			if exitCode := executor.GetPodExitCode(pod); pod.Status.Phase == corev1.PodFailed || exitCode != 0 {
				errorMessage = executor.GetPodErrorMessage(ctx, s.clientset, pod)
				break
			}
		}

		if errorMessage != "" {
			s.logger.Infow("execution pod failed with error message", "executionId", executionID, "message", execution.ExecutionResult.ErrorMessage)
			execution.ExecutionResult.Error()
			if execution.ExecutionResult.ErrorMessage != "" {
				execution.ExecutionResult.ErrorMessage += "\n"
			}

			execution.ExecutionResult.ErrorMessage += errorMessage
			err = s.resultRepository.UpdateResult(ctx, executionID, execution)
			if err != nil {
				s.logger.Errorf("update execution result returned an error %v while storing for execution id: %s", err, executionID)
				return err
			}
		}
	}

	return nil
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
	getAddrress := func(object metav1.Object) func(c context.Context, delay time.Duration) (string, error) {
		return func(c context.Context, delay time.Duration) (string, error) {
			return getServiceAdress(ctx, s.clientset, object)
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
			event := newWatcherEvent(testtrigger.EventCreated, service, testtrigger.ResourceService,
				withConditionsGetter(getConditions(service)), withAddressGetter(getAddrress(service)))
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
			event := newWatcherEvent(testtrigger.EventModified, newService, testtrigger.ResourceService,
				withConditionsGetter(getConditions(newService)), withAddressGetter(getAddrress(newService)))
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
			event := newWatcherEvent(testtrigger.EventDeleted, service, testtrigger.ResourceService,
				withConditionsGetter(getConditions(service)), withAddressGetter(getAddrress(service)))
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

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for created testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceTrigger, t.Name))
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

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updated testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceTrigger, t.Name))
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

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleted testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceTrigger, t.Name))
		},
	}
}

func (s *Service) testSuiteEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			testSuite, ok := obj.(*testsuitev3.TestSuite)
			if !ok {
				s.logger.Errorf("failed to process create testsuite event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(testSuite.CreationTimestamp.Time, s.watchFromDate) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create test suite: test suite %s/%s was created in the past",
					testSuite.Namespace, testSuite.Name,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding testsuite %s/%s",
				testSuite.Namespace, testSuite.Name,
			)
			s.addTestSuite(testSuite)

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for creating testsuite %s/%s",
				testSuite.Namespace, testSuite.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceTestsuite, testSuite.Name))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			testSuite, ok := newObj.(*testsuitev3.TestSuite)
			if !ok {
				s.logger.Errorf("failed to process update testsuite event due to it being an unexpected type, received type %+v", newObj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updating testsuite %s/%s",
				testSuite.Namespace, testSuite.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceTestsuite, testSuite.Name))
		},
		DeleteFunc: func(obj interface{}) {
			testSuite, ok := obj.(*testsuitev3.TestSuite)
			if !ok {
				s.logger.Errorf("failed to process delete testsuite event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleting testsuite %s/%s",
				testSuite.Namespace, testSuite.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceTestsuite, testSuite.Name))
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
					"trigger service: watcher component: no-op create test: test %s/%s was created in the past",
					test.Namespace, test.Name,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding test %s/%s",
				test.Namespace, test.Name,
			)
			s.addTest(test)

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for test %s/%s",
				test.Namespace, test.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceTest, test.Name))
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

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updating test %s/%s",
				test.Namespace, test.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceTest, test.Name))
		},
		DeleteFunc: func(obj interface{}) {
			test, ok := obj.(*testsv3.Test)
			if !ok {
				s.logger.Errorf("failed to process delete test event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleting test %s/%s",
				test.Namespace, test.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceTest, test.Name))
		},
	}
}

func (s *Service) executorEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			executor, ok := obj.(*executorv1.Executor)
			if !ok {
				s.logger.Errorf("failed to process create executor event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for executor %s/%s",
				executor.Namespace, executor.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceExecutor, executor.Name))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			executor, ok := newObj.(*executorv1.Executor)
			if !ok {
				s.logger.Errorf("failed to process update executor event due to it being an unexpected type, received type %+v", newObj)
				return
			}

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updating executor %s/%s",
				executor.Namespace, executor.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceExecutor, executor.Name))
		},
		DeleteFunc: func(obj interface{}) {
			executor, ok := obj.(*executorv1.Executor)
			if !ok {
				s.logger.Errorf("failed to process delete executor event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleting executor %s/%s",
				executor.Namespace, executor.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceExecutor, executor.Name))
		},
	}
}

func (s *Service) webhookEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			webhook, ok := obj.(*executorv1.Webhook)
			if !ok {
				s.logger.Errorf("failed to process create webhook event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for webhook %s/%s",
				webhook.Namespace, webhook.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceWebhook, webhook.Name))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			webhook, ok := newObj.(*executorv1.Webhook)
			if !ok {
				s.logger.Errorf("failed to process update webhook event due to it being an unexpected type, received type %+v", newObj)
				return
			}

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updating webhook %s/%s",
				webhook.Namespace, webhook.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceWebhook, webhook.Name))
		},
		DeleteFunc: func(obj interface{}) {
			webhook, ok := obj.(*executorv1.Webhook)
			if !ok {
				s.logger.Errorf("failed to process delete webhook event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleting webhook %s/%s",
				webhook.Namespace, webhook.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceWebhook, webhook.Name))
		},
	}
}

func (s *Service) testSourceEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			testSource, ok := obj.(*testsourcev1.TestSource)
			if !ok {
				s.logger.Errorf("failed to process create test source event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for test source %s/%s",
				testSource.Namespace, testSource.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventCreated, testkube.EventResourceTestsource, testSource.Name))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			testSource, ok := newObj.(*testsourcev1.TestSource)
			if !ok {
				s.logger.Errorf("failed to process update test source event due to it being an unexpected type, received type %+v", newObj)
				return
			}

			s.logger.Debugf(
				"trigger service: watcher component: emitting event for updating test source %s/%s",
				testSource.Namespace, testSource.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventUpdated, testkube.EventResourceTestsource, testSource.Name))
		},
		DeleteFunc: func(obj interface{}) {
			testSource, ok := obj.(*testsourcev1.TestSource)
			if !ok {
				s.logger.Errorf("failed to process delete test source event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emitting event for deleting test source %s/%s",
				testSource.Namespace, testSource.Name,
			)
			s.eventsBus.Publish(testkube.NewEvent(testkube.EventDeleted, testkube.EventResourceTestsource, testSource.Name))
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
