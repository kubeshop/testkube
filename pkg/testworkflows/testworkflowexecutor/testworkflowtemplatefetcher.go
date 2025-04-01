package testworkflowexecutor

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	TestWorkflowTemplateFetchParallelism = 10
)

type testWorkflowTemplateFetcher struct {
	client        testworkflowtemplateclient.TestWorkflowTemplateClient
	environmentId string
	cache         map[string]*testkube.TestWorkflowTemplate
	cacheMu       sync.RWMutex
}

func NewTestWorkflowTemplateFetcher(
	client testworkflowtemplateclient.TestWorkflowTemplateClient,
	environmentId string,
) *testWorkflowTemplateFetcher {
	return &testWorkflowTemplateFetcher{
		client:        client,
		environmentId: environmentId,
		cache:         make(map[string]*testkube.TestWorkflowTemplate),
	}
}

func (r *testWorkflowTemplateFetcher) SetCache(name string, tpl *testkube.TestWorkflowTemplate) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	if tpl == nil {
		delete(r.cache, name)
	} else {
		r.cache[name] = tpl
	}
}

func (r *testWorkflowTemplateFetcher) Prefetch(name string) error {
	name = testworkflowresolver.GetInternalTemplateName(name)
	r.cacheMu.RLock()
	if _, ok := r.cache[name]; ok {
		r.cacheMu.RUnlock()
		return nil
	}
	r.cacheMu.RUnlock()

	workflow, err := r.client.Get(context.Background(), r.environmentId, name)
	if err != nil {
		return errors.Wrapf(err, "cannot fetch Test Workflow Template by name: %s", name)
	}
	r.SetCache(name, workflow)
	return nil
}

func (r *testWorkflowTemplateFetcher) PrefetchMany(namesSet map[string]struct{}) error {
	// Internalize and dedupe names
	internalNames := make(map[string]struct{}, len(namesSet))
	for name := range namesSet {
		internalNames[testworkflowresolver.GetInternalTemplateName(name)] = struct{}{}
	}

	// Fetch all the requested templates
	var g errgroup.Group
	g.SetLimit(TestWorkflowTemplateFetchParallelism)
	for name := range internalNames {
		func(n string) {
			g.Go(func() error {
				return r.Prefetch(n)
			})
		}(name)
	}
	return g.Wait()
}

func (r *testWorkflowTemplateFetcher) Get(name string) (*testkube.TestWorkflowTemplate, error) {
	r.cacheMu.RLock()
	if r.cache[name] == nil {
		r.cacheMu.RUnlock()
		err := r.Prefetch(name)
		if err != nil {
			return nil, err
		}
		r.cacheMu.RLock()
	}
	tmpl := r.cache[name]
	r.cacheMu.RUnlock()
	return tmpl, nil
}

func (r *testWorkflowTemplateFetcher) GetMany(names map[string]struct{}) (map[string]*testkube.TestWorkflowTemplate, error) {
	results := make(map[string]*testkube.TestWorkflowTemplate, len(names))
	resultsMu := &sync.Mutex{}

	// Fetch all the requested templates
	var g errgroup.Group
	g.SetLimit(TestWorkflowTemplateFetchParallelism)
	for name := range names {
		func(n string) {
			g.Go(func() error {
				v, err := r.Get(n)
				if err != nil {
					return err
				}
				resultsMu.Lock()
				defer resultsMu.Unlock()
				results[v.Name] = v
				return nil
			})
		}(name)
	}
	err := g.Wait()

	return results, err
}
