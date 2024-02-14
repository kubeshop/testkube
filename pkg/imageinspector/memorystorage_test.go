package imageinspector

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	time1     = time.Now().UTC().Add(-4 * time.Minute)
	time2     = time.Now().UTC().Add(-2 * time.Minute)
	time3     = time.Now().UTC()
	noInfoPtr *Info
	info1     = Info{
		FetchedAt:  time1,
		Entrypoint: []string{"en", "try"},
		Cmd:        []string{"c", "md"},
		Shell:      "/bin/shell",
		WorkingDir: "some-wd",
	}
	info2 = Info{
		FetchedAt:  time2,
		Entrypoint: []string{"en", "try2"},
		Cmd:        []string{"c", "md2"},
		Shell:      "/bin/shell",
		WorkingDir: "some-wd",
	}
	info3 = Info{
		FetchedAt:  time3,
		Entrypoint: []string{"en", "try3"},
		Cmd:        []string{"c", "md3"},
		Shell:      "/bin/shell",
		WorkingDir: "some-wd",
	}
	req1 = RequestBase{
		Registry: "foo",
		Image:    "bar",
	}
	req1Copy = RequestBase{
		Registry: "foo",
		Image:    "bar",
	}
	req2 = RequestBase{
		Registry: "foo2",
		Image:    "bar2",
	}
	req3 = RequestBase{
		Registry: "foo3",
		Image:    "bar3",
	}
)

func TestMemoryStorageGetAndStore(t *testing.T) {
	m := NewMemoryStorage()
	err1 := m.Store(context.Background(), req1, info1)
	err2 := m.Store(context.Background(), req2, info2)
	v1, gErr1 := m.Get(context.Background(), req1)
	v2, gErr2 := m.Get(context.Background(), req2)
	v3, gErr3 := m.Get(context.Background(), req1Copy)
	v4, gErr4 := m.Get(context.Background(), req3)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, gErr1)
	assert.NoError(t, gErr2)
	assert.NoError(t, gErr3)
	assert.NoError(t, gErr4)
	assert.Equal(t, &info1, v1)
	assert.Equal(t, &info2, v2)
	assert.Equal(t, &info1, v3)
	assert.Equal(t, noInfoPtr, v4)
}

func TestMemoryStorageStoreManyAndGet(t *testing.T) {
	m := NewMemoryStorage()
	err1 := m.StoreMany(context.Background(), map[Hash]Info{
		hash(req1.Registry, req1.Image): info1,
		hash(req2.Registry, req2.Image): info2,
	})
	v1, gErr1 := m.Get(context.Background(), req1)
	v2, gErr2 := m.Get(context.Background(), req2)
	v3, gErr3 := m.Get(context.Background(), req1Copy)
	v4, gErr4 := m.Get(context.Background(), req3)
	assert.NoError(t, err1)
	assert.NoError(t, gErr1)
	assert.NoError(t, gErr2)
	assert.NoError(t, gErr3)
	assert.NoError(t, gErr4)
	assert.Equal(t, &info1, v1)
	assert.Equal(t, &info2, v2)
	assert.Equal(t, &info1, v3)
	assert.Equal(t, noInfoPtr, v4)
}

func TestMemoryStorageFillAndCopyTo(t *testing.T) {
	m := NewMemoryStorage()
	value := map[Hash]Info{
		hash(req1.Registry, req1.Image): info1,
		hash(req2.Registry, req2.Image): info2,
	}
	err1 := m.StoreMany(context.Background(), value)

	ctrl := gomock.NewController(t)
	mockStorage1 := NewMockStorageWithTransfer(ctrl)
	mockStorage2 := NewMockStorageWithTransfer(ctrl)
	mockStorage1.EXPECT().StoreMany(gomock.Any(), value)
	mockStorage2.EXPECT().StoreMany(gomock.Any(), value)
	err2 := m.CopyTo(context.Background(), mockStorage1, mockStorage2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}
