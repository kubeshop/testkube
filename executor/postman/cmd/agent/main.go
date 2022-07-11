package main

import (
	"os"

	"github.com/kubeshop/testkube/executor/postman/pkg/runner/newman"
	"github.com/kubeshop/testkube/pkg/executor/agent"
)

func main() {
	agent.Run(newman.NewNewmanRunner(), os.Args)
}
