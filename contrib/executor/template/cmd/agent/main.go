package main

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/template/pkg/runner"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	ctx := context.Background()
	agent.PreRun(ctx)
	defer agent.PostRun(ctx)
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize Template Executor environment variables: %v", err))
		os.Exit(1)
	}
	agent.Run(ctx, runner.NewRunner(params), os.Args)
}
