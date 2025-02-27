package agent

import (
	"context"
	"errors"

	"github.com/kubeshop/testkube/pkg/executor/output"
)

func GetDeprecatedLogStream(ctx context.Context, executionID string) (chan output.Output, error) {
	return nil, errors.New("deprecated features have been disabled")
}
