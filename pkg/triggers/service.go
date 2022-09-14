package triggers

import (
	"context"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	testtriggerclientsetv1 "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	v1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"time"
)

type Service struct {
	informers     *Informers
	triggers      []*testtriggersv1.TestTrigger
	started       time.Time
	triggerStatus map[StatusKey]*TriggerStatus
	tcs           testtriggerclientsetv1.Interface
	cs            *kubernetes.Clientset
	tsc           *testsuitesclientv2.TestSuitesClient
	tc            *testsclientv3.TestsClient
	tk            *v1.TestkubeAPI
	l             *zap.SugaredLogger
}

func NewService(
	cs *kubernetes.Clientset,
	tcs testtriggerclientsetv1.Interface,
	tsc *testsuitesclientv2.TestSuitesClient,
	tc *testsclientv3.TestsClient,
	tk *v1.TestkubeAPI,
	l *zap.SugaredLogger,
) *Service {
	return &Service{
		cs:            cs,
		tcs:           tcs,
		tsc:           tsc,
		tc:            tc,
		tk:            tk,
		l:             l,
		started:       time.Now(),
		triggers:      make([]*testtriggersv1.TestTrigger, 0),
		triggerStatus: make(map[StatusKey]*TriggerStatus),
	}
}

func (s *Service) Run(ctx context.Context) error {
	informers, err := s.createInformers(ctx)
	if err != nil {
		return err
	}

	s.l.Debugf("trigger service is starting pod informer")
	go informers.podInformer.Informer().Run(ctx.Done())
	s.l.Debugf("trigger service is starting deployment informer")
	go informers.deploymentInformer.Informer().Run(ctx.Done())
	s.l.Debugf("trigger service is starting testtrigger informer")
	go informers.testtriggerInformer.Informer().Run(ctx.Done())

	go s.runExecutionScraper(ctx)

	return nil
}

func (s *Service) AddTrigger(t *testtriggersv1.TestTrigger) {
	s.triggers = append(s.triggers, t)
	key := NewStatusKey(t.Namespace, t.Name)
	s.triggerStatus[key] = NewTriggerStatus()
}

func (s *Service) UpdateTrigger(target *testtriggersv1.TestTrigger) {
	for i, t := range s.triggers {
		if t.Namespace == target.Namespace && t.Name == target.Name {
			s.triggers[i] = target
			break
		}
	}
}

func (s *Service) RemoveTrigger(target *testtriggersv1.TestTrigger) {
	for i, t := range s.triggers {
		if t.Namespace == target.Namespace && t.Name == target.Name {
			s.triggers = append(s.triggers[:i], s.triggers[i+1:]...)

			break
		}
	}
	key := NewStatusKey(target.Namespace, target.Name)
	delete(s.triggerStatus, key)
}
