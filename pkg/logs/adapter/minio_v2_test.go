package adapter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

func generateWideLine(sizeKb int) string {
	b := strings.Builder{}
	for i := 0; i < sizeKb; i++ {
		b.WriteString(utils.RandAlphanum(1024))
	}

	return b.String()
}

func TestLogsV2Local(t *testing.T) {
	t.Skip("only local")
	ctx := context.Background()
	consumer, _ := NewMinioV2Adapter(cfg.StorageEndpoint, cfg.StorageAccessKeyID, cfg.StorageSecretAccessKey, cfg.StorageRegion, cfg.StorageToken, "test-1", cfg.StorageSSL, cfg.StorageSkipVerify, cfg.StorageCertFile, cfg.StorageKeyFile, cfg.StorageCAFile)
	consumer.WithNonEmptyPath("./")
	id := "test-bla"
	err := consumer.Init(ctx, id)
	assert.NoError(t, err)
	for i := 0; i < 10; i++ {
		fmt.Println("sending", i)
		consumer.Notify(ctx, id, events.Log{Time: time.Now(),
			Content: fmt.Sprintf("Test %d: %s", i, generateWideLine(200)),
			Type_:   "test", Source: strconv.Itoa(i)})
	}
	err = consumer.Stop(ctx, id)
	assert.NoError(t, err)
}
