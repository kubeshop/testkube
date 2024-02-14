package imageinspector

import (
	"context"
	"maps"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/pkg/configmap"
)

func mustMarshalInfo(v Info) string {
	s, e := marshalInfo(v)
	if e != nil {
		panic(e)
	}
	return s
}

func TestConfigMapStorageGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)): mustMarshalInfo(info2),
	}

	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)

	v1, err1 := m.Get(context.Background(), req1)
	assert.NoError(t, err1)
	assert.Equal(t, &info1, v1)
}

func TestConfigMapStorageGetEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)

	client.EXPECT().Get(gomock.Any(), "dummy").
		Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, "dummy"))

	v1, err1 := m.Get(context.Background(), req1)
	assert.NoError(t, err1)
	assert.Equal(t, noInfoPtr, v1)
}

func TestConfigMapStorageStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)): mustMarshalInfo(info1),
	}
	expected := map[string]string{
		string(hash(req2.Registry, req2.Image)): mustMarshalInfo(info2),
	}
	maps.Copy(expected, value)

	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)
	client.EXPECT().Apply(gomock.Any(), "dummy", expected).Return(nil)

	err1 := m.Store(context.Background(), req2, info2)
	assert.NoError(t, err1)
}

func TestConfigMapStorageStoreTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)):     mustMarshalInfo(info1),
		string(hash(req1.Registry+"A", req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
		string(hash(req2.Registry+"A", req2.Image)): mustMarshalInfo(info2),
	}
	initial := map[string]string{
		string(hash(req1.Registry, req1.Image)):     mustMarshalInfo(info1),
		string(hash(req1.Registry+"A", req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
		string(hash(req2.Registry+"A", req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)):     mustMarshalInfo(info3),
	}
	expected := map[string]string{
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
		string(hash(req2.Registry+"A", req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)):     mustMarshalInfo(info3),
	}

	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)
	client.EXPECT().Apply(gomock.Any(), "dummy", initial).Return(k8serrors.NewRequestEntityTooLargeError("test"))
	client.EXPECT().Apply(gomock.Any(), "dummy", expected).Return(nil)

	err1 := m.Store(context.Background(), req3, info3)
	assert.NoError(t, err1)
}

func TestConfigMapStorageStoreMany(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)): mustMarshalInfo(info1),
	}
	expected := map[string]string{
		string(hash(req2.Registry, req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)): mustMarshalInfo(info3),
	}
	maps.Copy(expected, value)

	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)
	client.EXPECT().Apply(gomock.Any(), "dummy", expected).Return(nil)

	err1 := m.StoreMany(context.Background(), map[Hash]Info{
		hash(req2.Registry, req2.Image): info2,
		hash(req3.Registry, req3.Image): info3,
	})
	assert.NoError(t, err1)
}

func TestConfigMapStorageStoreManyTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)):     mustMarshalInfo(info1),
		string(hash(req1.Registry+"A", req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
	}
	initial := map[string]string{
		string(hash(req1.Registry, req1.Image)):     mustMarshalInfo(info1),
		string(hash(req1.Registry+"A", req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
		string(hash(req2.Registry+"A", req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)):     mustMarshalInfo(info3),
	}
	expected := map[string]string{
		string(hash(req2.Registry, req2.Image)):     mustMarshalInfo(info2),
		string(hash(req2.Registry+"A", req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)):     mustMarshalInfo(info3),
	}

	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)
	client.EXPECT().Apply(gomock.Any(), "dummy", initial).Return(k8serrors.NewRequestEntityTooLargeError("test"))
	client.EXPECT().Apply(gomock.Any(), "dummy", expected).Return(nil)

	err1 := m.StoreMany(context.Background(), map[Hash]Info{
		hash(req2.Registry+"A", req2.Image): info2,
		hash(req3.Registry, req3.Image):     info3,
	})
	assert.NoError(t, err1)
}

func TestConfigMapStorageCopyTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := configmap.NewMockInterface(ctrl)
	s := NewMockStorageWithTransfer(ctrl)
	m := NewConfigMapStorage(client, "dummy", false)
	value := map[string]string{
		string(hash(req1.Registry, req1.Image)): mustMarshalInfo(info1),
		string(hash(req2.Registry, req2.Image)): mustMarshalInfo(info2),
		string(hash(req3.Registry, req3.Image)): mustMarshalInfo(info3),
	}
	expected := map[Hash]Info{
		hash(req1.Registry, req1.Image): info1,
		hash(req2.Registry, req2.Image): info2,
		hash(req3.Registry, req3.Image): info3,
	}
	client.EXPECT().Get(gomock.Any(), "dummy").Return(value, nil)
	s.EXPECT().StoreMany(gomock.Any(), expected).Return(nil)

	err1 := m.CopyTo(context.Background(), s)
	assert.NoError(t, err1)
}
