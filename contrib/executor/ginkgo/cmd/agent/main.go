package main

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/ginkgo/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	ctx := context.Background()
	ginkgo, err := runner.NewGinkgoRunner(ctx)
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize runner: %v", err))
		os.Exit(1)
	}
	agent.Run(ctx, ginkgo, os.Args)
}
