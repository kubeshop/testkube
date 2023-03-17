package main

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/ginkgo/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	ginkgo, err := runner.NewGinkgoRunner()
	if err != nil {
		output.PrintError(os.Stderr, fmt.Errorf("could not initialize runner: %w", err))
		os.Exit(1)
	}
	agent.Run(ginkgo, os.Args)
}
