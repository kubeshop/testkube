package triggers

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/scheduler"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	testkubeclientsetv1 "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

var defaultScraperInterval = 5 * time.Second

type Service struct {
	executor             ExecutorF
	scraperInterval      time.Duration
	triggers             []*testtriggersv1.TestTrigger
	started              time.Time
	triggerStatus        map[statusKey]*triggerStatus
	scheduler            *scheduler.Scheduler
	clientset            kubernetes.Interface
	testKubeClientset    testkubeclientsetv1.Interface
	testSuitesClient     testsuitesclientv2.Interface
	testsClient          testsclientv3.Interface
	resultRepository     result.Repository
	testResultRepository testresult.Repository
	logger               *zap.SugaredLogger
}

type Option func(*Service)

func NewService(
	scheduler *scheduler.Scheduler,
	clientset *kubernetes.Clientset,
	testTriggersClientset testkubeclientsetv1.Interface,
	testSuitesClient testsuitesclientv2.Interface,
	testsClient testsclientv3.Interface,
	resultRepository result.Repository,
	testResultRepository testresult.Repository,
	logger *zap.SugaredLogger,
	opts ...Option,
) *Service {
	s := &Service{
		scraperInterval:      defaultScraperInterval,
		scheduler:            scheduler,
		clientset:            clientset,
		testKubeClientset:    testTriggersClientset,
		testSuitesClient:     testSuitesClient,
		testsClient:          testsClient,
		resultRepository:     resultRepository,
		testResultRepository: testResultRepository,
		logger:               logger,
		started:              time.Now(),
		triggers:             make([]*testtriggersv1.TestTrigger, 0),
		triggerStatus:        make(map[statusKey]*triggerStatus),
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
	s.runWatcher(ctx)

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
