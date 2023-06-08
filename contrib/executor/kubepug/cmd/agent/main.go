package main

import (
	"context"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/kubepug/pkg/runner"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	ctx := context.Background()
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize Kubepug Executor environment variables: %v", err))
		os.Exit(1)
	}
	r, err := runner.NewRunner(ctx, params)
	if err != nil {
		log.Fatalf("%s could not run artillery tests: %s", ui.IconCross, err.Error())
	}
	agent.Run(ctx, r, os.Args)
}
