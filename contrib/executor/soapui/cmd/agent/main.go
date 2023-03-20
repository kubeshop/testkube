package main

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/soapui/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

func main() {
	r, err := runner.NewRunner()
	if err != nil {
		output.PrintError(os.Stderr, fmt.Errorf("could not initialize runner: %w", err))
		os.Exit(1)
	}

	agent.Run(r, os.Args)
}
