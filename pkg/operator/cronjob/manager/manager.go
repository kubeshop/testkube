package manager

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"

	configmapclient "github.com/kubeshop/testkube/pkg/operator/configmap"
	cronjobclient "github.com/kubeshop/testkube/pkg/operator/cronjob/client"
	namespaceclient "github.com/kubeshop/testkube/pkg/operator/namespace"
)

const (
	enableCronJobsFLagName = "enable-cron-jobs"
	reconciliationInterval = 30 * time.Second
)

//go:generate mockgen -destination=./mock_client.go -package=manager "github.com/kubeshop/testkube/pkg/operator/cronjob/manager" Interface
type Interface interface {
	IsNamespaceForNewArchitecture(namespace string) bool
	CleanForNewArchitecture(ctx context.Context) error
	Reconcile(ctx context.Context, log logr.Logger) error
}

// Manager provide methods to manage cronjobs
type Manager struct {
	namespaceClient   namespaceclient.Interface
	configMapClient   configmapclient.Interface
	cronJobClient     cronjobclient.Interface
	configMapName     string
	namespaceDetected sync.Map
}

// New is a method to create new cronjob manager
func New(namespaceClient namespaceclient.Interface, configMapClient configmapclient.Interface,
	cronJobClient cronjobclient.Interface, configMapName string) *Manager {
	return &Manager{
		namespaceClient: namespaceClient,
		configMapClient: configMapClient,
		cronJobClient:   cronJobClient,
		configMapName:   configMapName,
	}
}

func (m *Manager) IsNamespaceForNewArchitecture(namespace string) bool {
	_, ok := m.namespaceDetected.Load(namespace)
	return ok
}

func (m *Manager) checkNamespacesForNewArchitecture(ctx context.Context) (map[string]struct{}, error) {
	if m.configMapName == "" {
		return nil, nil
	}

	list, err := m.namespaceClient.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	namespaces := make(map[string]struct{})
	for _, namespace := range list.Items {
		data, err := m.configMapClient.Get(ctx, m.configMapName, namespace.Name)
		if err != nil {
			return nil, err
		}

		if data == nil {
			continue
		}

		if flag, ok := data[enableCronJobsFLagName]; ok && flag != "" {
			result, err := strconv.ParseBool(flag)
			if err != nil {
				return nil, err
			}

			if result {
				namespaces[namespace.Name] = struct{}{}
			}
		}
	}

	return namespaces, nil
}

// CleanForNewArchitecture is a method to clean cronjobs for new architecture
func (m *Manager) CleanForNewArchitecture(ctx context.Context) error {
	namespaces, err := m.checkNamespacesForNewArchitecture(ctx)
	if err != nil {
		return err
	}

	for namespace := range namespaces {
		if _, ok := m.namespaceDetected.Load(namespace); !ok {
			resources := []string{cronjobclient.TestResourceURI, cronjobclient.TestSuiteResourceURI, cronjobclient.TestWorkflowResourceURI}
			for _, resource := range resources {
				if err = m.cronJobClient.DeleteAll(ctx, getSelector(resource), namespace); err != nil {
					return err
				}
			}

			m.namespaceDetected.Store(namespace, struct{}{})
		}
	}

	m.namespaceDetected.Range(func(key any, value any) bool {
		if data, ok := key.(string); ok && namespaces != nil {
			if _, ok := namespaces[data]; !ok {
				m.namespaceDetected.Delete(data)
			}
		}

		return true
	})

	return nil
}

func (m *Manager) Reconcile(ctx context.Context, log logr.Logger) error {
	ticker := time.NewTicker(reconciliationInterval)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			if err := m.CleanForNewArchitecture(ctx); err != nil {
				log.Error(err, "unable to clean cron jobs for new architecture")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func getSelector(resource string) string {
	return fmt.Sprintf("testkube=%s", resource)
}
