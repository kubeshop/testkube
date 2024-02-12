package result

import (
	"context"
	"strings"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/storage"
)

func TestGetOutputSize(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	storageMock := storage.NewMockClient(mockCtrl)
	outputClient := NewMinioOutputRepository(storageMock, nil, "test-bucket")
	streamContent := "test line"
	storageMock.EXPECT().DownloadFileFromBucket(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(strings.NewReader(streamContent), minio.ObjectInfo{}, nil)
	size, err := outputClient.GetOutputSize(context.Background(), "test-id", "test-name", "test-suite-name")
	assert.Nil(t, err)
	assert.Equal(t, len(streamContent), size)

}
