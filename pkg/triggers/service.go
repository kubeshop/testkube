package triggers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/utils"

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

var (
	defaultScraperInterval    = 5 * time.Second
	defaultLeaseCheckInterval = 5 * time.Second
	defaultMaxLeaseDuration   = 1 * time.Minute
	defaultClusterID          = "testkube-api"
	defaultIdentifierFormat   = "testkube-api-%s"
)

type Service struct {
	informers            *k8sInformers
	leaseBackend         LeaseBackend
	identifier           string
	clusterID            string
	executor             ExecutorF
	scraperInterval      time.Duration
	leaseCheckInterval   time.Duration
	maxLeaseDuration     time.Duration
	watchFromDate        time.Time
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
	clientset kubernetes.Interface,
	testKubeClientset testkubeclientsetv1.Interface,
	testSuitesClient testsuitesclientv2.Interface,
	testsClient testsclientv3.Interface,
	resultRepository result.Repository,
	testResultRepository testresult.Repository,
	leaseBackend LeaseBackend,
	logger *zap.SugaredLogger,
	opts ...Option,
) *Service {
	identifier := fmt.Sprintf(defaultIdentifierFormat, utils.RandAlphanum(10))
	s := &Service{
		informers:            newK8sInformers(clientset, testKubeClientset),
		identifier:           identifier,
		clusterID:            defaultClusterID,
		scraperInterval:      defaultScraperInterval,
		leaseCheckInterval:   defaultLeaseCheckInterval,
		maxLeaseDuration:     defaultMaxLeaseDuration,
		scheduler:            scheduler,
		clientset:            clientset,
		testKubeClientset:    testKubeClientset,
		testSuitesClient:     testSuitesClient,
		testsClient:          testsClient,
		resultRepository:     resultRepository,
		testResultRepository: testResultRepository,
		leaseBackend:         leaseBackend,
		logger:               logger,
		watchFromDate:        time.Now(),
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

func WithIdentifier(id string) Option {
	return func(s *Service) {
		s.identifier = id
	}
}

func WithHostnameIdentifier() Option {
	return func(s *Service) {
		identifier, err := os.Hostname()
		if err == nil {
			s.identifier = identifier
		}
	}
}

func WithClusterID(id string) Option {
	return func(s *Service) {
		s.clusterID = id
	}
}

func WithWatchFromDate(from time.Time) Option {
	return func(s *Service) {
		s.watchFromDate = from
	}
}

func WithLeaseCheckerInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.leaseCheckInterval = interval
	}
}

func WithScraperInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.scraperInterval = interval
	}
}

func WithExecutor(executor ExecutorF) Option {
	return func(s *Service) {
		s.executor = executor
	}
}

func (s *Service) Run(ctx context.Context) {
	leaseChan := make(chan bool)

	go s.runLeaseChecker(ctx, leaseChan)

	go s.runWatcher(ctx, leaseChan)

	go s.runExecutionScraper(ctx)
}

func (s *Service) addTrigger(t *testtriggersv1.TestTrigger) {
	key := newStatusKey(t.Namespace, t.Name)
	s.triggerStatus[key] = newTriggerStatus(t)
}

func (s *Service) updateTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	if s.triggerStatus[key] != nil {
		s.triggerStatus[key].testTrigger = target
	} else {
		s.triggerStatus[key] = newTriggerStatus(target)
	}
}

func (s *Service) removeTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	delete(s.triggerStatus, key)
}
