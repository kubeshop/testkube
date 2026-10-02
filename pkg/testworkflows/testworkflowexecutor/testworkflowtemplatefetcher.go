package testworkflowexecutor

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

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
		display := testworkflowresolver.GetDisplayTemplateName(name)
		// The Kubernetes error names the template again, so a missing template gets one sentence.
		if k8serrors.IsNotFound(err) {
			return &templateNotFoundError{name: display, err: err}
		}
		return errors.Wrapf(err, "the template %q", display)
	}
	r.SetCache(name, template)
	return nil
}

// templateNotFoundError names a template that does not exist. It keeps the error of the client
// for errors.Is.
type templateNotFoundError struct {
	name string
	err  error
}

func (e *templateNotFoundError) Error() string {
	return fmt.Sprintf("the template %q does not exist", e.name)
}

func (e *templateNotFoundError) Unwrap() error { return e.err }

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
	v, ok := r.cache.Load(testworkflowresolver.GetInternalTemplateName(name))
	if !ok {
		err := r.Prefetch(name)
		if err != nil {
			return nil, err
		}
		v, ok = r.cache.Load(testworkflowresolver.GetInternalTemplateName(name))
		if !ok {
			return nil, errors.Errorf("unknown test workflow template %s", name)
		}
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
