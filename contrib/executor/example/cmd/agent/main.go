package main

import (
	"context"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/example/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
)

func main() {
	ctx := context.Background()
	agent.PreRun(ctx)
	defer agent.PostRun(ctx)
	agent.Run(ctx, runner.NewRunner(), os.Args)
}
