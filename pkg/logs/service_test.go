package logs

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/logs/adapter"

	"github.com/stretchr/testify/assert"
)

func TestLogsService_AddAdapter(t *testing.T) {

	t.Run("should add adapter", func(t *testing.T) {
		svc := LogsService{}

		svc.AddAdapter(adapter.NewDummyAdapter())
		svc.AddAdapter(adapter.NewDummyAdapter())
		svc.AddAdapter(adapter.NewDummyAdapter())
		svc.AddAdapter(adapter.NewDummyAdapter())

		assert.Equal(t, 4, len(svc.adapters))
	})

}
