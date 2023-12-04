package logs

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/logs/consumer"

	"github.com/stretchr/testify/assert"
)

func TestLogsService_AddAdapter(t *testing.T) {

	t.Run("should add adapter", func(t *testing.T) {
		svc := LogsService{}

		svc.AddAdapter(consumer.NewDummyAdapter())
		svc.AddAdapter(consumer.NewDummyAdapter())
		svc.AddAdapter(consumer.NewDummyAdapter())
		svc.AddAdapter(consumer.NewDummyAdapter())

		assert.Equal(t, 4, len(svc.adapters))
	})

}
