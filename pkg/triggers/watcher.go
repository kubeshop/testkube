package triggers

import (
	"context"
	"github.com/google/go-cmp/cmp"
	testtriggers_v1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/informers/externalversions"
	testtriggersinformerv1 "github.com/kubeshop/testkube-operator/pkg/informers/externalversions/testtrigger/v1"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	informersappsv1 "k8s.io/client-go/informers/apps/v1"
	informers_core_v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type EventType string
type ResourceType string
type Cause string

const (
	ResourcePod                       ResourceType = "pod"
	ResourceDeployment                ResourceType = "deployment"
	EventCreated                      EventType    = "created"
	EventModified                     EventType    = "modified"
	EventDeleted                      EventType    = "deleted"
	CauseDeploymentScaleUpdate        Cause        = "deployment_scale_update"
	CauseDeploymentImageUpdate        Cause        = "deployment_image_update"
	CauseDeploymentEnvUpdate          Cause        = "deployment_env_update"
	CauseDeploymentContainersModified Cause        = "deployment_containers_modified"
)

type Informers struct {
	podInformer         informers_core_v1.PodInformer
	deploymentInformer  informersappsv1.DeploymentInformer
	testtriggerInformer testtriggersinformerv1.TestTriggerInformer
}

func (s *Service) createInformers(ctx context.Context) (*Informers, error) {
	f := informers.NewSharedInformerFactory(s.cs, 0)
	podInformer := f.Core().V1().Pods()
	deploymentInformer := f.Apps().V1().Deployments()
	//daemonsetInformer := w.f.Apps().V1().DaemonSets()
	//statefulsetInformer := w.f.Apps().V1().StatefulSets()
	//serviceInformer := w.f.Core().V1().Services()
	//ingressInformer := w.f.Networking().V1().Ingresses()
	//eventInformer := w.f.Events().V1()

	testTriggerInformerFactory := externalversions.NewSharedInformerFactory(s.tcs, 0)
	testtriggerInformer := testTriggerInformerFactory.TestTrigger().V1().TestTriggers()

	podInformer.Informer().AddEventHandler(s.podEventHandler(ctx))
	deploymentInformer.Informer().AddEventHandler(s.deploymentEventHandler(ctx))
	testtriggerInformer.Informer().AddEventHandler(s.testtriggerEventHandler())

	return &Informers{
		podInformer:         podInformer,
		deploymentInformer:  deploymentInformer,
		testtriggerInformer: testtriggerInformer,
	}, nil
}

func (s *Service) podEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod, ok := obj.(*core_v1.Pod)
			if !ok {
				s.l.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(pod.CreationTimestamp.Time, s.started) {
				s.l.Debugf(
					"trigger service: watcher component: no-op create trigger: pod %s/%s was created in the past",
					pod.Namespace, pod.Name,
				)
				return
			}
			s.l.Debugf("trigger service: watcher component: emiting event: pod %s/%s created", pod.Namespace, pod.Name)
			event := newPodEvent(EventCreated, pod)
			if err := s.Match(ctx, event); err != nil {
				s.l.Errorf("event matcher returned an error while matching create pod event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*core_v1.Pod)
			if !ok {
				s.l.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.l.Debugf("trigger service: watcher component: emiting event: pod %s/%s deleted", pod.Namespace, pod.Name)
			event := newPodEvent(EventDeleted, pod)
			if err := s.Match(ctx, event); err != nil {
				s.l.Errorf("event matcher returned an error while matching delete pod event: %v", err)
			}
		},
	}
}

func (s *Service) deploymentEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			deployment, ok := obj.(*apps_v1.Deployment)
			if !ok {
				s.l.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(deployment.CreationTimestamp.Time, s.started) {
				s.l.Debugf(
					"trigger service: watcher component: no-op create trigger: deployment %s/%s was created in the past",
					deployment.Namespace, deployment.Name,
				)
				return
			}
			s.l.Debugf("emiting event: deployment %s/%s created", deployment.Namespace, deployment.Name)
			event := newDeploymentEvent(deployment, EventCreated, nil)
			if err := s.Match(ctx, event); err != nil {
				s.l.Errorf("event matcher returned an error while matching create deployment event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok := oldObj.(*apps_v1.Deployment)
			if !ok {
				s.l.Errorf(
					"failed to process update deployment event for old deployment due to it being an unexpected type, received type %+v",
					oldDeployment,
				)
				return
			}
			newDeployment, ok := newObj.(*apps_v1.Deployment)
			if !ok {
				s.l.Errorf(
					"failed to process update deployment event for new deployment due to it being an unexpected type, received type %+v",
					newDeployment,
				)
				return
			}
			if cmp.Equal(oldDeployment.Spec, newDeployment.Spec) {
				s.l.Debugf("trigger service: watcher component: no-op update trigger: deployment specs are equal")
				return
			}
			s.l.Debugf(
				"trigger service: watcher component: emiting event: deployment %s/%s updated",
				newDeployment.Namespace, newDeployment.Name,
			)
			causes := diffDeployments(oldDeployment, newDeployment)
			event := newDeploymentEvent(newDeployment, EventModified, causes)
			if err := s.Match(ctx, event); err != nil {
				s.l.Errorf("event matcher returned an error while matching update deployment event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deployment, ok := obj.(*apps_v1.Deployment)
			if !ok {
				s.l.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.l.Debugf("trigger service: watcher component: emiting event: deployment %s/%s deleted", deployment.Namespace, deployment.Name)
			event := newDeploymentEvent(deployment, EventDeleted, nil)
			if err := s.Match(ctx, event); err != nil {
				s.l.Errorf("event matcher returned an error while matching delete deployment event: %v", err)
			}
		},
	}
}

func (s *Service) testtriggerEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.l.Errorf("failed to process create testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.l.Debugf(
				"trigger service: watcher component: adding testtrigger %s/%s for resource %s and event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.AddTrigger(t)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			t, ok := newObj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.l.Errorf(
					"failed to process update testtrigger event for new testtrigger due to it being an unexpected type, received type %+v",
					newObj,
				)
				return
			}
			s.l.Debugf(
				"trigger service: watcher component: updating testtrigger %s/%s for resource %s and event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.UpdateTrigger(t)
		},
		DeleteFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.l.Errorf("failed to process delete testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.l.Debugf(
				"trigger service: watcher component: deleting testtrigger %s/%s for resource %s and event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.RemoveTrigger(t)
		},
	}
}
