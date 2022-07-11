package main

import (
	"os"

	"github.com/kubeshop/testkube/executor/artillery/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
)

func main() {
	agent.Run(runner.NewArtilleryRunner(), os.Args)
}
