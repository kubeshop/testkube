package k8sevent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sLoader(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	k8seventLoader := NewK8sEventLoader(clientset, "", nil)

	listeners, err := k8seventLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
