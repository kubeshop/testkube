package main

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"

	"github.com/kubeshop/testkube/contrib/executor/k6/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
)

func main() {
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize K6 Executor environment variables: %v", err))
		os.Exit(1)
	}
	agent.Run(context.Background(), runner.NewRunner(params), os.Args)
}
