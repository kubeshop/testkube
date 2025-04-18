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
	cache         sync.Map
}

func NewTestWorkflowTemplateFetcher(
	client testworkflowtemplateclient.TestWorkflowTemplateClient,
	environmentId string,
) *testWorkflowTemplateFetcher {
	return &testWorkflowTemplateFetcher{
		client:        client,
		environmentId: environmentId,
	}
}

func (r *testWorkflowTemplateFetcher) SetCache(name string, tpl *testkube.TestWorkflowTemplate) {
	if tpl == nil {
		r.cache.Delete(name)
	} else {
		r.cache.Store(name, tpl)
	}
}

func (r *testWorkflowTemplateFetcher) Prefetch(name string) error {
	name = testworkflowresolver.GetInternalTemplateName(name)
	if _, ok := r.cache.Load(name); ok {
		return nil
	}
	template, err := r.client.Get(context.Background(), r.environmentId, name)
	if err != nil {
		return errors.Wrapf(err, "cannot fetch Test Workflow Template by name: %s", name)
	}
	r.SetCache(name, template)
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
	v, ok := r.cache.Load(name)
	if !ok {
		err := r.Prefetch(name)
		if err != nil {
			return nil, err
		}
		v, _ = r.cache.Load(name)
	}
	return v.(*testkube.TestWorkflowTemplate), nil
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
