package main

import (
	"context"
	"log"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/artillery/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	ctx := context.Background()
	r, err := runner.NewArtilleryRunner(ctx)
	if err != nil {
		log.Fatalf("%s could not run artillery tests: %s", ui.IconCross, err.Error())
	}
	agent.Run(ctx, r, os.Args)
}
