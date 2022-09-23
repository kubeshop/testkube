package triggers

import (
	"context"

	"github.com/google/go-cmp/cmp"
	testtriggers_v1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/informers/externalversions"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (s *Service) runWatcher(ctx context.Context) {
	f := informers.NewSharedInformerFactory(s.clientset, 0)
	podInformer := f.Core().V1().Pods()
	deploymentInformer := f.Apps().V1().Deployments()
	// daemonsetInformer := w.f.Apps().V1().DaemonSets()
	// statefulsetInformer := w.f.Apps().V1().StatefulSets()
	// serviceInformer := w.f.Core().V1().Services()
	// ingressInformer := w.f.Networking().V1().Ingresses()
	// eventInformer := w.f.Events().V1()

	testkubeInformerFactory := externalversions.NewSharedInformerFactory(s.testKubeClientset, 0)
	testTriggerInformer := testkubeInformerFactory.Tests().V1().TestTriggers()

	podInformer.Informer().AddEventHandler(s.podEventHandler(ctx))
	deploymentInformer.Informer().AddEventHandler(s.deploymentEventHandler(ctx))
	testTriggerInformer.Informer().AddEventHandler(s.testtriggerEventHandler())

	s.logger.Debugf("trigger service: starting pod informer")
	go podInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting deployment informer")
	go deploymentInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting testtrigger informer")
	go testTriggerInformer.Informer().Run(ctx.Done())
}

func (s *Service) podEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(pod.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: pod %s/%s was created in the past",
					pod.Namespace, pod.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s created", pod.Namespace, pod.Name)
			event := newPodEvent(testtrigger.EventCreated, pod)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create pod event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s deleted", pod.Namespace, pod.Name)
			event := newPodEvent(testtrigger.EventDeleted, pod)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete pod event: %v", err)
			}
		},
	}
}

func (s *Service) deploymentEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(deployment.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: deployment %s/%s was created in the past",
					deployment.Namespace, deployment.Name,
				)
				return
			}
			s.logger.Debugf("emiting event: deployment %s/%s created", deployment.Namespace, deployment.Name)
			event := newDeploymentEvent(deployment, testtrigger.EventCreated, nil)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create deployment event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok := oldObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for old deployment due to it being an unexpected type, received type %+v",
					oldDeployment,
				)
				return
			}
			newDeployment, ok := newObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for new deployment due to it being an unexpected type, received type %+v",
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
			event := newDeploymentEvent(newDeployment, testtrigger.EventModified, causes)
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
			event := newDeploymentEvent(deployment, testtrigger.EventDeleted, nil)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete deployment event: %v", err)
			}
		},
	}
}

func (s *Service) testtriggerEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggers_v1.TestTrigger)
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
			t, ok := newObj.(*testtriggers_v1.TestTrigger)
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
			t, ok := obj.(*testtriggers_v1.TestTrigger)
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
