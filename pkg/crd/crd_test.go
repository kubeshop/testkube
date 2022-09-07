package crd

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/stretchr/testify/assert"
)

func TestGenerateYAML(t *testing.T) {

	t.Run("generate single CRD yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  events:\n  - start-test\n  uri: http://localhost\n  selector: app=backend\n"
		webhooks := []testkube.Webhook{
			{
				Name:      "name1",
				Namespace: "namespace1",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventStartTest},
				Selector:  "app=backend",
				Labels:    map[string]string{"key1": "value1"},
			},
		}

		// when
		result, err := GenerateYAML[testkube.Webhook](TemplateWebhook, webhooks)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("generate multiple CRDs yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  events:\n  - start-test\n  uri: http://localhost\n  selector: app=backend\n\n---\napiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name2\n  namespace: namespace2\n  labels:\n    key2: value2\nspec:\n  events:\n  - end-test\n  uri: http://localhost\n  selector: app=backend\n"
		webhooks := []testkube.Webhook{
			{
				Name:      "name1",
				Namespace: "namespace1",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventStartTest},
				Selector:  "app=backend",
				Labels:    map[string]string{"key1": "value1"},
			},
			{
				Name:      "name2",
				Namespace: "namespace2",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventEndTest},
				Selector:  "app=backend",
				Labels:    map[string]string{"key2": "value2"},
			},
		}

		// when
		result, err := GenerateYAML[testkube.Webhook](TemplateWebhook, webhooks)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

}
