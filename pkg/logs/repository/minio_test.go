package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/storage"
)

func TestRepository_MinioGet(t *testing.T) {
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

	storageClient.EXPECT().DownloadFileFromBucket(gomock.Any(), "bucket", "", "test-execution-1").Return(bytes.NewReader(data), nil)
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

			close(logs)

			for out := range logs {
				res = append(res, out.Log)
			}

			assert.Equal(t, tt.eventLogs, res)
		})
	}
}
