package imageinspector

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/pkg/secret"
)

func TestSecretFetcherGetExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client)

	expected := corev1.Secret{
		StringData: map[string]string{"key": "value"},
	}
	client.EXPECT().GetObject("dummy").Return(&expected, nil)

	result, err := fetcher.Get(context.Background(), "dummy")
	assert.NoError(t, err)
	assert.Equal(t, &expected, result)
}

func TestSecretFetcherGetCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client)

	expected := corev1.Secret{
		StringData: map[string]string{"key": "value"},
	}
	client.EXPECT().GetObject("dummy").Return(&expected, nil)

	result1, err1 := fetcher.Get(context.Background(), "dummy")
	result2, err2 := fetcher.Get(context.Background(), "dummy")
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, &expected, result1)
	assert.Equal(t, &expected, result2)
}

func TestSecretFetcherGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := secret.NewMockInterface(ctrl)
	fetcher := NewSecretFetcher(client)

	client.EXPECT().GetObject("dummy").Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, "dummy"))
	client.EXPECT().GetObject("dummy").Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, "dummy"))

	result1, err1 := fetcher.Get(context.Background(), "dummy")
	result2, err2 := fetcher.Get(context.Background(), "dummy")
	var noSecret *corev1.Secret
	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.True(t, k8serrors.IsNotFound(err1))
	assert.True(t, k8serrors.IsNotFound(err2))
	assert.Equal(t, noSecret, result1)
	assert.Equal(t, noSecret, result2)
}
