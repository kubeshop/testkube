package testsources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
)

func TestTestSources(t *testing.T) {
	var wClient *TestSourcesClient
	testTestSources := []*testsourcev1.TestSource{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsource1",
				Namespace: "test-ns",
			},
			Spec: testsourcev1.TestSourceSpec{
				Type_: "string",
				Data:  "test body",
			},
			Status: testsourcev1.TestSourceStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsource2",
				Namespace: "test-ns",
			},
			Spec: testsourcev1.TestSourceSpec{
				Type_: "string",
				Data:  "test body",
			},
			Status: testsourcev1.TestSourceStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsource3",
				Namespace: "test-ns",
			},
			Spec: testsourcev1.TestSourceSpec{
				Type_: "string",
				Data:  "test body",
			},
			Status: testsourcev1.TestSourceStatus{},
		},
	}

	t.Run("NewWebhooksClient", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder()

		groupVersion := schema.GroupVersion{Group: "tests.testkube.io", Version: "v1"}
		schemaBuilder := scheme.Builder{GroupVersion: groupVersion}
		schemaBuilder.Register(&testsourcev1.TestSourceList{})
		schemaBuilder.Register(&testsourcev1.TestSource{})
		schemaBuilder.Register(&corev1.Secret{})

		schema, err := schemaBuilder.Build()
		assert.NoError(t, err)
		assert.NotEmpty(t, schema)
		clientBuilder.WithScheme(schema)

		kClient := clientBuilder.Build()
		testSourceNamespace := "test-ns"
		wClient = NewClient(kClient, testSourceNamespace)
		assert.NotEmpty(t, wClient)
		assert.Equal(t, testSourceNamespace, wClient.namespace)
	})
	t.Run("TestCreate", func(t *testing.T) {
		t.Run("Create new test sources", func(t *testing.T) {
			for _, w := range testTestSources {
				created, err := wClient.Create(w)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, created.Name)

				res, err := wClient.Get(w.Name)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, res.Name)
			}
		})
	})
	t.Run("TestList", func(t *testing.T) {
		t.Run("List without selector", func(t *testing.T) {
			l, err := wClient.List("")
			assert.NoError(t, err)
			assert.Equal(t, len(testTestSources), len(l.Items))
		})
	})
	t.Run("TestGet", func(t *testing.T) {
		t.Run("Get testsource with empty name", func(t *testing.T) {
			t.Parallel()
			_, err := wClient.Get("")
			assert.Error(t, err)
		})

		t.Run("Get testsource with non existent name", func(t *testing.T) {
			t.Parallel()
			_, err := wClient.Get("no-testsource")
			assert.Error(t, err)
		})

		t.Run("Get existing testsource", func(t *testing.T) {
			res, err := wClient.Get(testTestSources[0].Name)
			assert.NoError(t, err)
			assert.Equal(t, testTestSources[0].Name, res.Name)
		})
	})
	t.Run("TestDelete", func(t *testing.T) {
		t.Run("Delete items", func(t *testing.T) {
			for _, testsource := range testTestSources {
				w, err := wClient.Get(testsource.Name)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, testsource.Name)

				err = wClient.Delete(testsource.Name)
				assert.NoError(t, err)

				_, err = wClient.Get(testsource.Name)
				assert.Error(t, err)
			}
		})

		t.Run("Delete non-existent item", func(t *testing.T) {
			_, err := wClient.Get("no-testsource")
			assert.Error(t, err)

			err = wClient.Delete("no-testsource")
			assert.Error(t, err)
		})
	})
}
