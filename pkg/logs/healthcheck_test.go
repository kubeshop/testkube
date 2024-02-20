package logs

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/log"
)

func TestLogsService_RunHealthcheckHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	svc := LogsService{log: log.DefaultLogger}
	svc.WithRandomPort()
	go svc.RunHealthCheckHandler(ctx)
	go svc.RunGRPCServer(ctx, nil)
	defer svc.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/health", svc.httpAddress))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
