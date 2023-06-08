package main

import (
	"context"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"

	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/runner/newman"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	ctx := context.Background()
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Errorf("could not initialize Postman Executor environment variables: %v", err))
		os.Exit(1)
	}
	r, err := newman.NewNewmanRunner(ctx, params)
	if err != nil {
		log.Fatalf("%s could not run Postman tests: %s", ui.IconCross, err.Error())
	}
	agent.Run(context.Background(), r, os.Args)
}
