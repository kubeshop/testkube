package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/scraper/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	ctx := context.Background()
	r, err := runner.NewRunner(ctx)
	if err != nil {
		output.PrintError(os.Stderr, fmt.Errorf("could not initialize runner: %w", err))
		os.Exit(1)
	}
	agent.Run(ctx, r, os.Args)
}
