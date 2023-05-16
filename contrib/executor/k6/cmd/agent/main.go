package main

import (
	"context"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/k6/pkg/runner"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	ctx := context.Background()
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize K6 Executor environment variables: %v", err))
		os.Exit(1)
	}
	r, err := runner.NewRunner(ctx, params)
	if err != nil {
		log.Fatalf("%s Could not run cURL tests: %s", ui.IconCross, err.Error())
	}

	agent.Run(ctx, r, os.Args)
}
