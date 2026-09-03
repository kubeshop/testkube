package localrunner

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultJobTTL            = time.Hour
	cleanupWaitTimeout       = 45 * time.Second
	cleanupPollInterval      = 250 * time.Millisecond
	staleRunThreshold        = 24 * time.Hour
	localNamespaceLabelValue = "true"
)

// ResourceManager owns only resources explicitly labelled by localrunner. It
// is intentionally unable to remove a namespace or select by object name.
type ResourceManager struct {
	client    kubernetes.Interface
	namespace string
}

func NewResourceManager(client kubernetes.Interface, namespace string) *ResourceManager {
	return &ResourceManager{client: client, namespace: namespace}
}

func (m *ResourceManager) Namespace() string { return m.namespace }

// EnsureNamespace creates the developer namespace only when it is absent. A
// namespace may have existed before this run and is never deleted by cleanup.
func (m *ResourceManager) EnsureNamespace(ctx context.Context) error {
	_, err := m.client.CoreV1().Namespaces().Get(ctx, m.namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get namespace %q: %w", m.namespace, err)
	}
	_, err = m.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   m.namespace,
			Labels: map[string]string{LocalLabel: localNamespaceLabelValue},
		},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %q: %w", m.namespace, err)
	}
	return nil
}

// Labels returns the exact ownership labels used on every local-run resource.
// Component must be a valid Kubernetes label value to keep selectors safe.
func Labels(runID, component string) (map[string]string, error) {
	if runID == "" {
		return nil, UsageError("local run ID must not be empty")
	}
	if component == "" {
		return nil, UsageError("local component must not be empty")
	}
	if messages := validation.IsValidLabelValue(runID); len(messages) > 0 {
		return nil, UsageError("local run ID %q is not Kubernetes-safe: %s", runID, strings.Join(messages, "; "))
	}
	if messages := validation.IsValidLabelValue(component); len(messages) > 0 {
		return nil, UsageError("local component %q is not Kubernetes-safe: %s", component, strings.Join(messages, "; "))
	}
	return map[string]string{
		LocalLabel:          localLabelValue,
		LocalRunIDLabel:     runID,
		LocalComponentLabel: component,
	}, nil
}

func localSelector(runID string) string {
	return fmt.Sprintf("%s=%s,%s=%s", LocalLabel, localLabelValue, LocalRunIDLabel, runID)
}

// Clean deletes exactly one local run's resources. It intentionally leaves any
// user-created Secret, ConfigMap, Service, or retained run that lacks both
// ownership labels untouched. It is idempotent.
func (m *ResourceManager) Clean(ctx context.Context, runID string) error {
	if _, err := Labels(runID, "cleanup"); err != nil {
		return err
	}
	if _, err := m.client.CoreV1().Namespaces().Get(ctx, m.namespace, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get namespace %q before local cleanup: %w", m.namespace, err)
	}
	selector := localSelector(runID)
	foreground := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{PropagationPolicy: &foreground}
	listOptions := metav1.ListOptions{LabelSelector: selector}

	var errs []error
	deleteListed := func(kind string, list func() ([]string, error), deleteOne func(string) error) {
		names, err := list()
		if err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("list local %s for run %q: %w", kind, runID, err))
			}
			return
		}
		for _, name := range names {
			if err := deleteOne(name); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete local %s %q for run %q: %w", kind, name, runID, err))
			}
		}
	}
	// Delete Jobs first so their Pods are not recreated while auxiliary
	// resources are being removed. Pod deletion covers a partially-created Job
	// and an interrupted relay upload.
	deleteListed("jobs", func() ([]string, error) {
		objects, err := m.client.BatchV1().Jobs(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedJobs(objects.Items), nil
	}, func(name string) error {
		return m.client.BatchV1().Jobs(m.namespace).Delete(ctx, name, deleteOptions)
	})
	deleteListed("pods", func() ([]string, error) {
		objects, err := m.client.CoreV1().Pods(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedPods(objects.Items), nil
	}, func(name string) error {
		return m.client.CoreV1().Pods(m.namespace).Delete(ctx, name, deleteOptions)
	})
	deleteListed("services", func() ([]string, error) {
		objects, err := m.client.CoreV1().Services(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedServices(objects.Items), nil
	}, func(name string) error {
		return m.client.CoreV1().Services(m.namespace).Delete(ctx, name, deleteOptions)
	})
	deleteListed("persistent volume claims", func() ([]string, error) {
		objects, err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedPVCs(objects.Items), nil
	}, func(name string) error {
		return m.client.CoreV1().PersistentVolumeClaims(m.namespace).Delete(ctx, name, deleteOptions)
	})
	deleteListed("config maps", func() ([]string, error) {
		objects, err := m.client.CoreV1().ConfigMaps(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedConfigMaps(objects.Items), nil
	}, func(name string) error {
		return m.client.CoreV1().ConfigMaps(m.namespace).Delete(ctx, name, deleteOptions)
	})
	deleteListed("secrets", func() ([]string, error) {
		objects, err := m.client.CoreV1().Secrets(m.namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		return namedSecrets(objects.Items), nil
	}, func(name string) error {
		return m.client.CoreV1().Secrets(m.namespace).Delete(ctx, name, deleteOptions)
	})
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	cleanupCtx, cancel := context.WithTimeout(ctx, cleanupWaitTimeout)
	defer cancel()
	for {
		remaining, err := m.runResources(cleanupCtx, selector)
		if err != nil {
			return err
		}
		if len(remaining) == 0 {
			return nil
		}
		select {
		case <-cleanupCtx.Done():
			return fmt.Errorf("waiting for exact local resources for run %q to be deleted: %s: %w", runID, strings.Join(remaining, ", "), cleanupCtx.Err())
		case <-time.After(cleanupPollInterval):
		}
	}
}

// StaleRunIDs reports local-labelled resources older than the warning age. It
// deliberately checks every resource type local cleanup owns: a completed Job
// can disappear through its one-hour TTL while a failed relay Service or a
// processor-generated ConfigMap remains. It does not mutate anything; the
// caller can print exact clean commands.
func (m *ResourceManager) StaleRunIDs(ctx context.Context, now time.Time) ([]string, error) {
	unique := map[string]struct{}{}
	observe := func(object metav1.Object) {
		runID := object.GetLabels()[LocalRunIDLabel]
		created := object.GetCreationTimestamp()
		if runID == "" || created.IsZero() || created.Add(staleRunThreshold).After(now) {
			return
		}
		unique[runID] = struct{}{}
	}
	selector := metav1.ListOptions{LabelSelector: LocalLabel + "=" + localLabelValue}
	lists := []struct {
		kind string
		list func() error
	}{
		{kind: "jobs", list: func() error {
			items, err := m.client.BatchV1().Jobs(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
		{kind: "pods", list: func() error {
			items, err := m.client.CoreV1().Pods(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
		{kind: "services", list: func() error {
			items, err := m.client.CoreV1().Services(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
		{kind: "persistent volume claims", list: func() error {
			items, err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
		{kind: "config maps", list: func() error {
			items, err := m.client.CoreV1().ConfigMaps(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
		{kind: "secrets", list: func() error {
			items, err := m.client.CoreV1().Secrets(m.namespace).List(ctx, selector)
			if err != nil {
				return err
			}
			for i := range items.Items {
				observe(&items.Items[i])
			}
			return nil
		}},
	}
	for _, resource := range lists {
		if err := resource.list(); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("list local %s for stale-run warning: %w", resource.kind, err)
		}
	}
	result := make([]string, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func (m *ResourceManager) runResources(ctx context.Context, selector string) ([]string, error) {
	type resourceList struct {
		kind string
		list func() ([]string, error)
	}
	items := []resourceList{
		{kind: "job", list: func() ([]string, error) {
			objects, err := m.client.BatchV1().Jobs(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedJobs(objects.Items), err
		}},
		{kind: "pod", list: func() ([]string, error) {
			objects, err := m.client.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedPods(objects.Items), err
		}},
		{kind: "service", list: func() ([]string, error) {
			objects, err := m.client.CoreV1().Services(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedServices(objects.Items), err
		}},
		{kind: "pvc", list: func() ([]string, error) {
			objects, err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedPVCs(objects.Items), err
		}},
		{kind: "configmap", list: func() ([]string, error) {
			objects, err := m.client.CoreV1().ConfigMaps(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedConfigMaps(objects.Items), err
		}},
		{kind: "secret", list: func() ([]string, error) {
			objects, err := m.client.CoreV1().Secrets(m.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			return namedSecrets(objects.Items), err
		}},
	}
	var result []string
	for _, item := range items {
		names, err := item.list()
		if err != nil {
			return nil, fmt.Errorf("list local %s resources: %w", item.kind, err)
		}
		for _, name := range names {
			result = append(result, item.kind+"/"+name)
		}
	}
	sort.Strings(result)
	return result, nil
}

func namedJobs(items []batchv1.Job) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}
func namedPods(items []corev1.Pod) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}
func namedServices(items []corev1.Service) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}
func namedPVCs(items []corev1.PersistentVolumeClaim) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}
func namedConfigMaps(items []corev1.ConfigMap) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}
func namedSecrets(items []corev1.Secret) []string {
	return objectNames(len(items), func(i int) string { return items[i].Name })
}

func objectNames(length int, name func(int) string) []string {
	result := make([]string, 0, length)
	for i := 0; i < length; i++ {
		result = append(result, name(i))
	}
	return result
}
