package main

import (
	"context"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	ctx := context.Background()
	runner, err := runner.NewRunner(ctx)
	if err != nil {
		ui.Err(err)
	}
	agent.Run(ctx, runner, os.Args)
}
