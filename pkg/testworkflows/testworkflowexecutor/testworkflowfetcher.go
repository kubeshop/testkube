package testworkflowexecutor

import (
	"context"
	"maps"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	TestWorkflowFetchParallelism = 10
)

type testWorkflowFetcher struct {
	client           testworkflowclient.TestWorkflowClient
	environmentId    string
	cache            sync.Map
	prefetchedLabels []map[string]string
}

func NewTestWorkflowFetcher(client testworkflowclient.TestWorkflowClient, environmentId string) *testWorkflowFetcher {
	return &testWorkflowFetcher{
		client:        client,
		environmentId: environmentId,
	}
}

func (r *testWorkflowFetcher) PrefetchByLabelSelector(labels map[string]string) error {
	if containsSameMap(r.prefetchedLabels, labels) {
		return nil
	}
	workflows, err := r.client.List(context.Background(), r.environmentId, testworkflowclient.ListOptions{Labels: labels})
	if err != nil {
		return errors.Wrapf(err, "cannot fetch Test Workflows by label selector: %v", labels)
	}
	for i := range workflows {
		r.cache.Store(workflows[i].Name, &workflows[i])
	}
	r.prefetchedLabels = append(r.prefetchedLabels, labels)
	return nil
}

func (r *testWorkflowFetcher) PrefetchByName(name string) error {
	if _, ok := r.cache.Load(name); ok {
		return nil
	}
	workflow, err := r.client.Get(context.Background(), r.environmentId, name)
	if err != nil {
		return errors.Wrapf(err, "cannot fetch Test Workflow by name: %s", name)
	}
	r.cache.Store(name, workflow)
	return nil
}

func (r *testWorkflowFetcher) PrefetchMany(selectors []*cloud.ScheduleResourceSelector) error {
	// Categorize selectors
	names := make(map[string]struct{})
	labels := make([]map[string]string, 0)
	for i := range selectors {
		if selectors[i].Name == "" {
			if !containsSameMap(labels, selectors[i].Labels) {
				labels = append(labels, selectors[i].Labels)
			}
		} else {
			names[selectors[i].Name] = struct{}{}
		}
	}

	// Fetch firstly by the label selector, as it is more likely to conflict with others
	g := errgroup.Group{}
	g.SetLimit(TestWorkflowFetchParallelism)
	for i := range labels {
		func(m map[string]string) {
			g.Go(func() error {
				return r.PrefetchByLabelSelector(labels[i])
			})
		}(labels[i])
	}
	err := g.Wait()
	if err != nil {
		return err
	}

	// Fetch the rest by name
	g = errgroup.Group{}
	g.SetLimit(TestWorkflowFetchParallelism)
	for name := range names {
		func(n string) {
			g.Go(func() error {
				return r.PrefetchByName(n)
			})
		}(name)
	}
	return g.Wait()
}

func (r *testWorkflowFetcher) GetByName(name string) (*testkube.TestWorkflow, error) {
	v, ok := r.cache.Load(name)
	if !ok {
		err := r.PrefetchByName(name)
		if err != nil {
			return nil, err
		}
		v, _ = r.cache.Load(name)
	}
	return v.(*testkube.TestWorkflow), nil
}

func (r *testWorkflowFetcher) GetByLabelSelector(labels map[string]string) ([]*testkube.TestWorkflow, error) {
	if !containsSameMap(r.prefetchedLabels, labels) {
		err := r.PrefetchByLabelSelector(labels)
		if err != nil {
			return nil, err
		}
	}
	result := make([]*testkube.TestWorkflow, 0)
	r.cache.Range(func(_, workflow any) bool {
		for k := range labels {
			if workflow.(*testkube.TestWorkflow).Labels[k] != labels[k] {
				return true
			}
		}
		result = append(result, workflow.(*testkube.TestWorkflow))
		return true
	})
	return result, nil
}

func (r *testWorkflowFetcher) Get(selector *cloud.ScheduleResourceSelector) ([]*testkube.TestWorkflow, error) {
	if selector.Name == "" {
		return r.GetByLabelSelector(selector.Labels)
	}
	v, err := r.GetByName(selector.Name)
	if err != nil {
		return nil, err
	}
	return []*testkube.TestWorkflow{v}, nil
}

func (r *testWorkflowFetcher) Names() []string {
	names := make([]string, 0)
	r.cache.Range(func(name, _ any) bool {
		names = append(names, name.(string))
		return true
	})
	return names
}

func (r *testWorkflowFetcher) TemplateNames() map[string]struct{} {
	result := make(map[string]struct{})
	r.cache.Range(func(_, workflow any) bool {
		// TODO: avoid converting to CRD
		maps.Copy(result, testworkflowresolver.ListTemplates(testworkflows.MapAPIToKube(workflow.(*testkube.TestWorkflow))))
		return true
	})
	return result
}

func containsSameMap[T comparable, U comparable](s []map[T]U, v map[T]U) bool {
	for i := range s {
		if len(s[i]) != len(v) {
			continue
		}
		equal := true
		for k := range s[i] {
			if x, ok := v[k]; !ok || x != s[i][k] {
				equal = false
				break
			}
		}
		if equal {
			return true
		}
	}
	return false
}
