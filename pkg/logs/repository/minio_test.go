package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/storage"
)

func TestRepository_MinioGetLogV2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storageClient := storage.NewMockClient(mockCtrl)
	ctx := context.TODO()

	var data []byte

	eventLog1 := events.Log{
		Content: "storage logs 1",
		Source:  events.SourceJobPod,
		Version: string(events.LogVersionV2),
	}

	b, err := json.Marshal(eventLog1)
	assert.NoError(t, err)

	data = append(data, b...)
	data = append(data, []byte("\n")...)

	eventLog2 := events.Log{
		Content: "storage logs 2",
		Source:  events.SourceJobPod,
		Version: string(events.LogVersionV2),
	}

	b, err = json.Marshal(eventLog2)
	assert.NoError(t, err)

	data = append(data, b...)
	data = append(data, []byte("\n")...)

	storageClient.EXPECT().DownloadFileFromBucket(gomock.Any(), "bucket", "", "test-execution-1").
		Return(bytes.NewReader(data), minio.ObjectInfo{}, nil)
	r := NewMinioRepository(storageClient, "bucket")

	tests := []struct {
		name      string
		eventLogs []events.Log
	}{
		{
			name:      "Test getting logs from minio",
			eventLogs: []events.Log{eventLog1, eventLog2},
		},
	}

	var res []events.Log
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := r.Get(ctx, "test-execution-1")
			assert.NoError(t, err)

			for out := range logs {
				res = append(res, out.Log)
			}

			assert.Equal(t, tt.eventLogs, res)
		})
	}
}

func TestRepository_MinioGetLogsV1(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storageClient := storage.NewMockClient(mockCtrl)
	ctx := context.TODO()

	var data []byte

	contentLog1 := "storage logs 1"
	contentLog2 := "storage logs 2"
	output := result.ExecutionOutput{
		Id:            "id",
		Name:          "execution-name",
		TestName:      "test-name",
		TestSuiteName: "testsuite-name",
		Output:        contentLog1 + "\n" + contentLog2,
	}

	data, err := json.Marshal(output)
	assert.NoError(t, err)

	current := time.Now()
	storageClient.EXPECT().DownloadFileFromBucket(gomock.Any(), "bucket", "", "test-execution-1").
		Return(bytes.NewReader(data), minio.ObjectInfo{LastModified: current}, nil)
	r := NewMinioRepository(storageClient, "bucket")

	tests := []struct {
		name      string
		eventLogs []events.Log
	}{
		{
			name: "Test getting logs from minio",
			eventLogs: []events.Log{
				{
					Time:    current,
					Content: contentLog1,
					Version: string(events.LogVersionV1),
				},
				{
					Time:    current,
					Content: contentLog2,
					Version: string(events.LogVersionV1),
				},
			},
		},
	}

	var res []events.Log
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := r.Get(ctx, "test-execution-1")
			assert.NoError(t, err)

			for out := range logs {
				res = append(res, out.Log)
			}

			assert.Equal(t, tt.eventLogs, res)
		})
	}
}
