package imageinspector

import (
	"context"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestInspectorInspect(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	storage1 := NewMockStorageWithTransfer(ctrl)
	storage2 := NewMockStorageWithTransfer(ctrl)
	inspector := NewInspector("default.io", infos, secrets, storage1, storage2)

	sec := corev1.Secret{StringData: map[string]string{"foo": "bar"}}
	req := RequestBase{Registry: "regname.io", Image: "imgname"}
	resolvedReq := RequestBase{Image: "regname.io/imgname"}
	storage1.EXPECT().Get(gomock.Any(), resolvedReq).Return(nil, nil)
	storage2.EXPECT().Get(gomock.Any(), resolvedReq).Return(nil, nil)
	storage1.EXPECT().Get(gomock.Any(), req).Return(nil, nil)
	storage2.EXPECT().Get(gomock.Any(), req).Return(nil, nil)
	secrets.EXPECT().Get(gomock.Any(), "secname").Return(&sec, nil)
	infos.EXPECT().Fetch(gomock.Any(), req.Registry, req.Image, []corev1.Secret{sec}).Return(&info1, nil)

	storage1.EXPECT().Store(gomock.Any(), resolvedReq, info1).Return(nil)
	storage2.EXPECT().Store(gomock.Any(), resolvedReq, info1).Return(nil)

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
	inspector := NewInspector("default.io", infos, secrets, storage1, storage2)

	req := RequestBase{Registry: "regname.io", Image: "imgname"}
	resolvedReq := RequestBase{Image: "regname.io/imgname"}
	storage1.EXPECT().Get(gomock.Any(), resolvedReq).Return(&info1, nil)

	v, err := inspector.Inspect(context.Background(), req.Registry, req.Image, corev1.PullIfNotPresent, []string{"secname"})
	assert.NoError(t, err)
	assert.Equal(t, &info1, v)

	// Wait until asynchronous storage will be done
	<-time.After(10 * time.Millisecond)
}

func TestInspector_ResolveName_NoDefault_NoOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	inspector := NewInspector("", infos, secrets)

	assert.Equal(t, "image:1.2.3", inspector.ResolveName("", "image:1.2.3"))
	assert.Equal(t, "repo/image:1.2.3", inspector.ResolveName("", "repo/image:1.2.3"))
	assert.Equal(t, "docker.io/image:1.2.3", inspector.ResolveName("", "docker.io/image:1.2.3"))
	assert.Equal(t, "ghcr.io/image:1.2.3", inspector.ResolveName("", "ghcr.io/image:1.2.3"))
	assert.Equal(t, "docker.io/repo/image:1.2.3", inspector.ResolveName("", "docker.io/repo/image:1.2.3"))
	assert.Equal(t, "ghcr.io/repo/image:1.2.3", inspector.ResolveName("", "ghcr.io/repo/image:1.2.3"))
}

func TestInspector_ResolveName_Default_NoOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	inspector := NewInspector("default.io", infos, secrets)

	assert.Equal(t, "default.io/image:1.2.3", inspector.ResolveName("", "image:1.2.3"))
	assert.Equal(t, "default.io/repo/image:1.2.3", inspector.ResolveName("", "repo/image:1.2.3"))
	assert.Equal(t, "docker.io/image:1.2.3", inspector.ResolveName("", "docker.io/image:1.2.3"))
	assert.Equal(t, "ghcr.io/image:1.2.3", inspector.ResolveName("", "ghcr.io/image:1.2.3"))
	assert.Equal(t, "docker.io/repo/image:1.2.3", inspector.ResolveName("", "docker.io/repo/image:1.2.3"))
	assert.Equal(t, "ghcr.io/repo/image:1.2.3", inspector.ResolveName("", "ghcr.io/repo/image:1.2.3"))
}

func TestInspector_ResolveName_NoDefault_Override(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	inspector := NewInspector("", infos, secrets)

	assert.Equal(t, "default.io/image:1.2.3", inspector.ResolveName("default.io", "image:1.2.3"))
	assert.Equal(t, "default.io/repo/image:1.2.3", inspector.ResolveName("default.io", "repo/image:1.2.3"))
	assert.Equal(t, "docker.io/image:1.2.3", inspector.ResolveName("default.io", "docker.io/image:1.2.3"))
	assert.Equal(t, "ghcr.io/image:1.2.3", inspector.ResolveName("default.io", "ghcr.io/image:1.2.3"))
	assert.Equal(t, "docker.io/repo/image:1.2.3", inspector.ResolveName("default.io", "docker.io/repo/image:1.2.3"))
	assert.Equal(t, "ghcr.io/repo/image:1.2.3", inspector.ResolveName("default.io", "ghcr.io/repo/image:1.2.3"))
}

func TestInspector_ResolveName_Default_Override(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	inspector := NewInspector("default.io", infos, secrets)

	assert.Equal(t, "default.io/image:1.2.3", inspector.ResolveName("default.io", "image:1.2.3"))
	assert.Equal(t, "default.io/repo/image:1.2.3", inspector.ResolveName("default.io", "repo/image:1.2.3"))
	assert.Equal(t, "docker.io/image:1.2.3", inspector.ResolveName("default.io", "docker.io/image:1.2.3"))
	assert.Equal(t, "ghcr.io/image:1.2.3", inspector.ResolveName("default.io", "ghcr.io/image:1.2.3"))
	assert.Equal(t, "docker.io/repo/image:1.2.3", inspector.ResolveName("default.io", "docker.io/repo/image:1.2.3"))
	assert.Equal(t, "ghcr.io/repo/image:1.2.3", inspector.ResolveName("default.io", "ghcr.io/repo/image:1.2.3"))
}

func TestInspector_ResolveName_CustomDefault_NoOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	infos := NewMockInfoFetcher(ctrl)
	secrets := NewMockSecretFetcher(ctrl)
	inspector := NewInspector("custom-registry:443", infos, secrets)

	assert.Equal(t, "custom-registry:443/repo/image:1.2.3", inspector.ResolveName("", "custom-registry:443/repo/image:1.2.3"))
}
