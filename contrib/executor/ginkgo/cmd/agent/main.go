package main

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/ginkgo/pkg/runner"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	ctx := context.Background()
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize Ginkgo Executor environment variables: %v", err))
		os.Exit(1)
	}
	ginkgo, err := runner.NewGinkgoRunner(ctx, params)
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize runner: %v", err))
		os.Exit(1)
	}
	agent.Run(ctx, ginkgo, os.Args)
}
