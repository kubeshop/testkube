package manager

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metaav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configmapclient "github.com/kubeshop/testkube/pkg/operator/configmap"
	cronjobclient "github.com/kubeshop/testkube/pkg/operator/cronjob/client"
	namespaceclient "github.com/kubeshop/testkube/pkg/operator/namespace"
)

func Test_CleanForNewArchitecture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigMapClient := configmapclient.NewMockInterface(mockCtrl)
	mockCronJobClient := cronjobclient.NewMockInterface(mockCtrl)
	mockNamespaceClient := namespaceclient.NewMockInterface(mockCtrl)
	mg := New(mockNamespaceClient, mockConfigMapClient, mockCronJobClient, "configmap")

	mockNamespaceList := corev1.NamespaceList{Items: []corev1.Namespace{
		{
			ObjectMeta: metaav1.ObjectMeta{
				Name: "testkube",
			},
		},
	}}
	mockNamespaceClient.EXPECT().ListAll(ctx, "").Return(&mockNamespaceList, nil).Times(1)

	mockConfigMap := map[string]string{
		"enable-cron-jobs": "true",
	}
	mockConfigMapClient.EXPECT().Get(ctx, "configmap", "testkube").Return(mockConfigMap, nil).Times(1)

	mockCronJobClient.EXPECT().DeleteAll(ctx, gomock.Any(), "testkube").Times(3)

	err := mg.CleanForNewArchitecture(ctx)
	assert.NoError(t, err)
}
