package main

import (
	"context"
	"log"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/runner/newman"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	r, err := newman.NewNewmanRunner()
	if err != nil {
		log.Fatalf("%s could not run Postman tests: %s", ui.IconCross, err.Error())
	}
	agent.Run(context.Background(), r, os.Args)
}
