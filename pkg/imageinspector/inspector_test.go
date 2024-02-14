package imageinspector

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestInspectorInspect(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	storage1 := NewMockStorageWithTransfer(ctrl)
	storage2 := NewMockStorageWithTransfer(ctrl)
	inspector := NewInspector("default", infos, secrets, storage1, storage2)

	sec := corev1.Secret{StringData: map[string]string{"foo": "bar"}}
	req := RequestBase{Registry: "regname", Image: "imgname"}
	storage1.EXPECT().Get(gomock.Any(), req).Return(nil, nil)
	storage2.EXPECT().Get(gomock.Any(), req).Return(nil, nil)
	secrets.EXPECT().Get(gomock.Any(), "secname").Return(&sec, nil)
	infos.EXPECT().Fetch(gomock.Any(), req.Registry, req.Image, []corev1.Secret{sec}).Return(&info1, nil)

	storage1.EXPECT().Store(gomock.Any(), req, info1).Return(nil)
	storage2.EXPECT().Store(gomock.Any(), req, info1).Return(nil)

	v, err := inspector.Inspect(context.Background(), req.Registry, req.Image, corev1.PullIfNotPresent, []string{"secname"})
	assert.NoError(t, err)
	assert.Equal(t, &info1, v)

	// Wait until asynchronous storage will be done
	<-time.After(10 * time.Millisecond)
}

func TestInspectorInspectWithCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	storage1 := NewMockStorageWithTransfer(ctrl)
	storage2 := NewMockStorageWithTransfer(ctrl)
	inspector := NewInspector("default", infos, secrets, storage1, storage2)

	req := RequestBase{Registry: "regname", Image: "imgname"}
	storage1.EXPECT().Get(gomock.Any(), req).Return(&info1, nil)

	v, err := inspector.Inspect(context.Background(), req.Registry, req.Image, corev1.PullIfNotPresent, []string{"secname"})
	assert.NoError(t, err)
	assert.Equal(t, &info1, v)

	// Wait until asynchronous storage will be done
	<-time.After(10 * time.Millisecond)
}
