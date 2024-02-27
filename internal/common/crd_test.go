package common

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
)

var (
	time1    = time.Now().UTC()
	testBare = testsv3.Test{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
		Spec: testsv3.TestSpec{
			Description: "some-description",
		},
	}
	testWithCreationTimestamp = testsv3.Test{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-name",
			CreationTimestamp: metav1.Time{Time: time1},
		},
		Spec: testsv3.TestSpec{
			Description: "some-description",
		},
	}
	testWrongOrder = testsv3.Test{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
		// Use keys that are not alphabetically ordered
		Spec: testsv3.TestSpec{
			Schedule:    "abc",
			Name:        "example-name",
			Description: "some-description",
		},
	}
	testMessyData = testsv3.Test{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
			ManagedFields: []metav1.ManagedFieldsEntry{
				{
					Manager:     "some-manager",
					Operation:   "some-operation",
					APIVersion:  "v1",
					FieldsType:  "blah",
					Subresource: "meh",
				},
			},
		},
		// Use keys that are not alphabetically ordered
		Spec: testsv3.TestSpec{
			Description: "some-description",
		},
	}
)

func TestSerializeCRDNoMutations(t *testing.T) {
	value := testBare.DeepCopy()
	_, _ = SerializeCRD(value, SerializeOptions{
		CleanMeta:             true,
		OmitCreationTimestamp: true,
		Kind:                  "Test",
		GroupVersion:          &testsv3.GroupVersion,
	})

	assert.Equal(t, value.TypeMeta, testBare.TypeMeta)
	assert.Equal(t, value.ObjectMeta, testBare.ObjectMeta)
}

func TestSerializeCRD(t *testing.T) {
	b, err := SerializeCRD(testBare.DeepCopy(), SerializeOptions{})
	b2, err2 := SerializeCRD(testWithCreationTimestamp.DeepCopy(), SerializeOptions{OmitCreationTimestamp: true})
	want := strings.TrimSpace(`
metadata:
  name: test-name
spec:
  description: some-description
status: {}
`)
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
	assert.NoError(t, err2)
	assert.Equal(t, want+"\n", string(b2))
}

func TestSerializeCRDWithCreationTimestamp(t *testing.T) {
	b, err := SerializeCRD(testWithCreationTimestamp.DeepCopy(), SerializeOptions{})
	want := strings.TrimSpace(`
metadata:
  name: test-name
  creationTimestamp: "%s"
spec:
  description: some-description
status: {}
`)
	want = fmt.Sprintf(want, time1.Format(time.RFC3339))
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
}

func TestSerializeCRDWithMessyData(t *testing.T) {
	b, err := SerializeCRD(testMessyData.DeepCopy(), SerializeOptions{})
	b2, err2 := SerializeCRD(testMessyData.DeepCopy(), SerializeOptions{CleanMeta: true})
	want := strings.TrimSpace(`
metadata:
  name: test-name
  managedFields:
  - manager: some-manager
    operation: some-operation
    apiVersion: v1
    fieldsType: blah
    subresource: meh
spec:
  description: some-description
status: {}
`)
	want2 := strings.TrimSpace(`
metadata:
  name: test-name
spec:
  description: some-description
status: {}
`)
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
	assert.NoError(t, err2)
	assert.Equal(t, want2+"\n", string(b2))
}

func TestSerializeCRDKeepOrder(t *testing.T) {
	b, err := SerializeCRD(*testWrongOrder.DeepCopy(), SerializeOptions{})
	want := strings.TrimSpace(`
metadata:
  name: test-name
spec:
  name: example-name
  description: some-description
  schedule: abc
status: {}
`)
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
}

func TestSerializeCRDs(t *testing.T) {
	b, err := SerializeCRDs([]testsv3.Test{
		*testWrongOrder.DeepCopy(),
		*testBare.DeepCopy(),
	}, SerializeOptions{})
	want := strings.TrimSpace(`
metadata:
  name: test-name
spec:
  name: example-name
  description: some-description
  schedule: abc
status: {}
---
metadata:
  name: test-name
spec:
  description: some-description
status: {}
`)
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
}

func TestSerializeCRDsFullCleanup(t *testing.T) {
	list := testsv3.TestList{
		Items: []testsv3.Test{
			*testWrongOrder.DeepCopy(),
			*testBare.DeepCopy(),
			*testWithCreationTimestamp.DeepCopy(),
		},
	}
	b, err := SerializeCRDs(list.Items, SerializeOptions{
		CleanMeta:             true,
		OmitCreationTimestamp: true,
		Kind:                  "Test",
		GroupVersion:          &testsv3.GroupVersion,
	})
	want := strings.TrimSpace(`
kind: Test
apiVersion: tests.testkube.io/v3
metadata:
  name: test-name
spec:
  name: example-name
  description: some-description
  schedule: abc
status: {}
---
kind: Test
apiVersion: tests.testkube.io/v3
metadata:
  name: test-name
spec:
  description: some-description
status: {}
---
kind: Test
apiVersion: tests.testkube.io/v3
metadata:
  name: test-name
spec:
  description: some-description
status: {}
`)
	assert.NoError(t, err)
	assert.Equal(t, want+"\n", string(b))
}
