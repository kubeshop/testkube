package testsuiteexecutions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
)

func TestTestSuiteExecutions(t *testing.T) {
	var tseClient *TestSuiteExecutionsClient
	testTestSuiteExecutions := []*testsuiteexecutionv1.TestSuiteExecution{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsuiteexecution1",
				Namespace: "test-ns",
			},
			Spec:   testsuiteexecutionv1.TestSuiteExecutionSpec{},
			Status: testsuiteexecutionv1.TestSuiteExecutionStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsuiteexecution2",
				Namespace: "test-ns",
			},
			Spec:   testsuiteexecutionv1.TestSuiteExecutionSpec{},
			Status: testsuiteexecutionv1.TestSuiteExecutionStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-testsuiteexecution3",
				Namespace: "test-ns",
			},
			Spec:   testsuiteexecutionv1.TestSuiteExecutionSpec{},
			Status: testsuiteexecutionv1.TestSuiteExecutionStatus{},
		},
	}

	t.Run("NewTestSuiteExecutionsClient", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder()

		groupVersion := schema.GroupVersion{Group: "tests.testkube.io", Version: "v1"}
		schemaBuilder := scheme.Builder{GroupVersion: groupVersion}
		schemaBuilder.Register(&testsuiteexecutionv1.TestSuiteExecutionList{})
		schemaBuilder.Register(&testsuiteexecutionv1.TestSuiteExecution{})

		schema, err := schemaBuilder.Build()
		assert.NoError(t, err)
		assert.NotEmpty(t, schema)
		clientBuilder.WithScheme(schema)

		kClient := clientBuilder.Build()
		testNamespace := "test-ns"
		tseClient = NewClient(kClient, testNamespace)
		assert.NotEmpty(t, tseClient)
		assert.Equal(t, testNamespace, tseClient.namespace)
	})
	t.Run("TestSuiteExecutionCreate", func(t *testing.T) {
		t.Run("Create new testsuiteexecutions", func(t *testing.T) {
			for _, te := range testTestSuiteExecutions {
				created, err := tseClient.Create(te)
				assert.NoError(t, err)
				assert.Equal(t, te.Name, created.Name)

				res, err := tseClient.Get(te.ObjectMeta.Name)
				assert.NoError(t, err)
				assert.Equal(t, te.Name, res.Name)
			}
		})
	})
	t.Run("TestSuiteExecutionUpdate", func(t *testing.T) {
		t.Run("Update new testsuiteexecutions", func(t *testing.T) {
			for _, te := range testTestSuiteExecutions {
				res, err := tseClient.Get(te.ObjectMeta.Name)
				assert.NoError(t, err)
				assert.Equal(t, te.Name, res.Name)

				updated, err := tseClient.Update(te)
				assert.NoError(t, err)
				assert.Equal(t, te.Name, updated.Name)
			}
		})
	})
	t.Run("TestSuiteExecutionGet", func(t *testing.T) {
		t.Run("Get testsuiteexecution with empty name", func(t *testing.T) {
			t.Parallel()
			_, err := tseClient.Get("")
			assert.Error(t, err)
		})

		t.Run("Get testsuiteexecution with non existent name", func(t *testing.T) {
			t.Parallel()
			_, err := tseClient.Get("no-testsuiteexecution")
			assert.Error(t, err)
		})

		t.Run("Get existing testsuiteexecution", func(t *testing.T) {
			res, err := tseClient.Get(testTestSuiteExecutions[0].Name)
			assert.NoError(t, err)
			assert.Equal(t, testTestSuiteExecutions[0].Name, res.Name)
		})
	})
	t.Run("TestSuiteExecutionDelete", func(t *testing.T) {
		t.Run("Delete items", func(t *testing.T) {
			for _, testsuiteexecution := range testTestSuiteExecutions {
				te, err := tseClient.Get(testsuiteexecution.Name)
				assert.NoError(t, err)
				assert.Equal(t, te.Name, testsuiteexecution.Name)

				err = tseClient.Delete(testsuiteexecution.Name)
				assert.NoError(t, err)

				_, err = tseClient.Get(testsuiteexecution.Name)
				assert.Error(t, err)
			}
		})

		t.Run("Delete non-existent item", func(t *testing.T) {
			_, err := tseClient.Get("no-testsuiteexecution")
			assert.Error(t, err)

			err = tseClient.Delete("no-testsuiteexecution")
			assert.Error(t, err)
		})
	})
}
