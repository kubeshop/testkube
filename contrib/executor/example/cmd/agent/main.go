package main

import (
	"context"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/example/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
)

func main() {
	agent.Run(context.Background(), runner.NewRunner(), os.Args)
}
