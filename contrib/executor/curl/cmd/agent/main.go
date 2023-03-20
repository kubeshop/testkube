package main

import (
	"log"
	"os"

	"github.com/kubeshop/testkube/contrib/executor/curl/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	r, err := runner.NewCurlRunner()
	if err != nil {
		log.Fatalf("%s Could not run cURL tests: %s", ui.IconCross, err.Error())
	}

	agent.Run(r, os.Args)
}
