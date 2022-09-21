package triggers

import (
	"context"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	testtriggerclientsetv1 "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	v1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

var defaultScraperInterval = 5 * time.Second

type Service struct {
	executor        ExecutorF
	scraperInterval time.Duration
	triggers        []*testtriggersv1.TestTrigger
	started         time.Time
	triggerStatus   map[statusKey]*triggerStatus
	tcs             testtriggerclientsetv1.Interface
	cs              *kubernetes.Clientset
	tsc             *testsuitesclientv2.TestSuitesClient
	tc              *testsclientv3.TestsClient
	tk              *v1.TestkubeAPI
	trr             result.Repository
	tsrr            testresult.Repository
	l               *zap.SugaredLogger
}

type Option func(*Service)

func NewService(
	cs *kubernetes.Clientset,
	tcs testtriggerclientsetv1.Interface,
	tsc *testsuitesclientv2.TestSuitesClient,
	tc *testsclientv3.TestsClient,
	tk *v1.TestkubeAPI,
	trr result.Repository,
	tsrr testresult.Repository,
	l *zap.SugaredLogger,
	opts ...Option,
) *Service {
	s := &Service{
		scraperInterval: defaultScraperInterval,
		cs:              cs,
		tcs:             tcs,
		tsc:             tsc,
		tc:              tc,
		tk:              tk,
		trr:             trr,
		tsrr:            tsrr,
		l:               l,
		started:         time.Now(),
		triggers:        make([]*testtriggersv1.TestTrigger, 0),
		triggerStatus:   make(map[statusKey]*triggerStatus),
	}
	if s.executor == nil {
		s.executor = s.execute
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) WithScraperInterval(interval time.Duration) {
	s.scraperInterval = interval
}

func (s *Service) WithExecutor(executor ExecutorF) {
	s.executor = executor
}

func (s *Service) Run(ctx context.Context) error {
	informers, err := s.createInformers(ctx)
	if err != nil {
		return err
	}

	s.l.Debugf("trigger service: starting pod informer")
	go informers.podInformer.Informer().Run(ctx.Done())
	s.l.Debugf("trigger service: starting deployment informer")
	go informers.deploymentInformer.Informer().Run(ctx.Done())
	s.l.Debugf("trigger service: starting testtrigger informer")
	go informers.testTriggerInformer.Informer().Run(ctx.Done())

	go s.runExecutionScraper(ctx)

	return nil
}

func (s *Service) addTrigger(t *testtriggersv1.TestTrigger) {
	s.triggers = append(s.triggers, t)
	key := newStatusKey(t.Namespace, t.Name)
	s.triggerStatus[key] = newTriggerStatus()
}

func (s *Service) updateTrigger(target *testtriggersv1.TestTrigger) {
	for i, t := range s.triggers {
		if t.Namespace == target.Namespace && t.Name == target.Name {
			s.triggers[i] = target
			break
		}
	}
}

func (s *Service) removeTrigger(target *testtriggersv1.TestTrigger) {
	for i, t := range s.triggers {
		if t.Namespace == target.Namespace && t.Name == target.Name {
			s.triggers = append(s.triggers[:i], s.triggers[i+1:]...)

			break
		}
	}
	key := newStatusKey(target.Namespace, target.Name)
	delete(s.triggerStatus, key)
}
