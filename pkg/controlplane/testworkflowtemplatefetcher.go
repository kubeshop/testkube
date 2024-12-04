package controlplane

import (
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

type testWorkflowTemplateFetcher struct {
	client           testworkflowsclientv1.TestWorkflowTemplatesInterface
	cache            map[string]*testworkflowsv1.TestWorkflowTemplate
	prefetchedLabels []map[string]string
}

func NewTestWorkflowTemplateFetcher(
	client testworkflowsclientv1.TestWorkflowTemplatesInterface,
) *testWorkflowTemplateFetcher {
	return &testWorkflowTemplateFetcher{
		client: client,
		cache:  make(map[string]*testworkflowsv1.TestWorkflowTemplate),
	}
}

func (r *testWorkflowTemplateFetcher) Prefetch(name string) error {
	name = testworkflowresolver.GetInternalTemplateName(name)
	if _, ok := r.cache[name]; ok {
		return nil
	}
	workflow, err := r.client.Get(name)
	if err != nil {
		return errors.Wrapf(err, "cannot fetch Test Workflow Template by name: %s", name)
	}
	r.cache[name] = workflow
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
	g.SetLimit(10)
	for name := range internalNames {
		func(n string) {
			g.Go(func() error {
				return r.Prefetch(n)
			})
		}(name)
	}
	return g.Wait()
}

func (r *testWorkflowTemplateFetcher) Get(name string) (*testworkflowsv1.TestWorkflowTemplate, error) {
	if r.cache[name] == nil {
		err := r.Prefetch(name)
		if err != nil {
			return nil, err
		}
	}
	return r.cache[name], nil
}

func (r *testWorkflowTemplateFetcher) GetMany(names map[string]struct{}) (map[string]*testworkflowsv1.TestWorkflowTemplate, error) {
	results := make(map[string]*testworkflowsv1.TestWorkflowTemplate, len(names))
	resultsMu := &sync.Mutex{}

	// Fetch all the requested templates
	var g errgroup.Group
	g.SetLimit(10)
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
