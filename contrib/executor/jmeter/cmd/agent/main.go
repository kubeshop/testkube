package main

import (
	"os"

	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	runner, err := runner.NewRunner()
	if err != nil {
		ui.Err(err)
	}
	agent.Run(runner, os.Args)
}
