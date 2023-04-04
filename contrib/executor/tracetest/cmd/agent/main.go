package main

import (
	"log"
	"os"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	r, err := runner.NewRunner()
	if err != nil {
		log.Fatalf("%s could not run Tracetest tests: %s", ui.IconCross, err.Error())
	}
	agent.Run(r, os.Args)
}
