package testworkflowexecutor

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
)

func TestTestWorkflowTemplateFetcher_ConcurrentAccess(t *testing.T) {
	// Create mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock client
	mockClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(ctrl)

	// Create test templates
	templates := make([]*testkube.TestWorkflowTemplate, 10)
	for i := 0; i < 10; i++ {
		templates[i] = &testkube.TestWorkflowTemplate{
			Name:        "template-" + string(rune('A'+i)),
			Description: "description-" + string(rune('A'+i)),
			Labels:      map[string]string{"key": "value"},
			Spec:        &testkube.TestWorkflowTemplateSpec{},
		}
	}

	// Setup mock expectations
	for _, tmpl := range templates {
		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", tmpl.Name).
			Return(tmpl, nil).
			AnyTimes()
	}

	// Create fetcher
	fetcher := NewTestWorkflowTemplateFetcher(mockClient, "test-env")

	// Test concurrent prefetch
	t.Run("concurrent prefetch", func(t *testing.T) {
		names := make(map[string]struct{})
		for _, tmpl := range templates {
			names[tmpl.Name] = struct{}{}
		}

		// Launch multiple goroutines to prefetch templates
		err := fetcher.PrefetchMany(names)
		assert.NoError(t, err)

		// Verify all templates were fetched
		for _, tmpl := range templates {
			fetched, err := fetcher.Get(tmpl.Name)
			assert.NoError(t, err)
			assert.Equal(t, tmpl.Name, fetched.Name)
			assert.Equal(t, tmpl.Description, fetched.Description)
			assert.Equal(t, tmpl.Labels, fetched.Labels)
		}
	})

	// Test concurrent get
	t.Run("concurrent get", func(t *testing.T) {
		names := make(map[string]struct{})
		for _, tmpl := range templates {
			names[tmpl.Name] = struct{}{}
		}

		// Launch multiple goroutines to get templates
		results, err := fetcher.GetMany(names)
		assert.NoError(t, err)
		assert.Len(t, results, len(templates))

		// Verify all templates were retrieved correctly
		for _, tmpl := range templates {
			fetched, ok := results[tmpl.Name]
			assert.True(t, ok)
			assert.Equal(t, tmpl.Name, fetched.Name)
			assert.Equal(t, tmpl.Description, fetched.Description)
			assert.Equal(t, tmpl.Labels, fetched.Labels)
		}
	})

	// Test concurrent set and get
	t.Run("concurrent set and get", func(t *testing.T) {
		// Create a new template
		newTmpl := &testkube.TestWorkflowTemplate{
			Name:        "new-template",
			Description: "new-description",
			Labels:      map[string]string{"key": "value"},
			Spec:        &testkube.TestWorkflowTemplateSpec{},
		}

		// Setup mock expectations for the new template
		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", newTmpl.Name).
			Return(newTmpl, nil).
			AnyTimes()

		// Launch goroutines to set and get the template concurrently
		done := make(chan struct{})
		go func() {
			for i := 0; i < 100; i++ {
				fetcher.SetCache(newTmpl.Name, newTmpl)
			}
			close(done)
		}()

		// Try to get the template while it's being set
		for i := 0; i < 100; i++ {
			fetched, err := fetcher.Get(newTmpl.Name)
			assert.NoError(t, err)
			assert.Equal(t, newTmpl.Name, fetched.Name)
			assert.Equal(t, newTmpl.Description, fetched.Description)
			assert.Equal(t, newTmpl.Labels, fetched.Labels)
		}

		<-done // Wait for set operations to complete
	})
}
