package imageinspector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
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
	// Only the versioned key is read, so a stale entry under the resolved name or the image alone
	// is never used.
	resolvedReq := RequestBase{Registry: resolvedCacheVersion, Image: "regname.io/imgname"}
	storage1.EXPECT().Get(gomock.Any(), resolvedReq).Return(nil, nil)
	storage2.EXPECT().Get(gomock.Any(), resolvedReq).Return(nil, nil)
	secrets.EXPECT().Get(gomock.Any(), "secname").Return(&sec, nil)
	infos.EXPECT().Fetch(gomock.Any(), "", resolvedReq.Image, []corev1.Secret{sec}).Return(&info1, nil)

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
	resolvedReq := RequestBase{Registry: resolvedCacheVersion, Image: "regname.io/imgname"}
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

func TestInspector_Inspect_Error(t *testing.T) {
	tests := []struct {
		name            string
		defaultRegistry string
		wantFetched     string
		info            *Info
		err             error
		secretErr       error
		want            string
	}{
		{
			name:        "an image without a default registry keeps its name",
			wantFetched: "imgname",
			err:         errors.New("no such host"),
			want:        `the image "imgname" cannot be read: no such host`,
		},
		{
			name:            "the image is fetched from the default registry that the pod pulls from",
			defaultRegistry: "default.io",
			wantFetched:     "default.io/imgname",
			err:             errors.New("no such host"),
			want:            `the image "default.io/imgname" cannot be read: no such host`,
		},
		{
			name:        "no details from the registry",
			wantFetched: "imgname",
			want:        `the image "imgname" cannot be read: the registry returned no details`,
		},
		{
			name:      "a pull secret that cannot be read names the secret",
			secretErr: errors.New(`secrets "regcred" not found`),
			want:      `the image "imgname" cannot be read: the pull secret "regcred" cannot be read: secrets "regcred" not found`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			infos := NewMockInfoFetcher(ctrl)
			secrets := NewMockSecretFetcher(ctrl)
			inspector := NewInspector(tt.defaultRegistry, infos, secrets)
			var secretNames []string
			if tt.secretErr != nil {
				secretNames = []string{"regcred"}
				secrets.EXPECT().Get(gomock.Any(), "regcred").Return(nil, tt.secretErr)
			} else {
				infos.EXPECT().Fetch(gomock.Any(), "", tt.wantFetched, gomock.Any()).Return(tt.info, tt.err)
			}

			_, err := inspector.Inspect(context.Background(), "", "imgname", corev1.PullAlways, secretNames)
			assert.EqualError(t, err, tt.want)
		})
	}
}
