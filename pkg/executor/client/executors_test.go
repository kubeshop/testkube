package client

import (
	"fmt"
	"testing"

	"github.com/kubeshop/kubtest-operator/client"
	executorscr "github.com/kubeshop/kubtest-operator/client/executors"
)

func TestExecutors(t *testing.T) {
	t.Skip("move it to some e2e test suite on cluster when ready")

	// TODO create resources

	// TODO test

	kubeClient := client.GetClient()
	executorsClient := executorscr.NewClient(kubeClient)
	execs := NewExecutors(executorsClient)

	postmanExec, err := execs.Get("postman/collection")
	fmt.Printf("Postman\n")
	fmt.Printf("%+v\n", err)
	fmt.Printf("%+v\n", postmanExec)

	curlExec, err := execs.Get("curl/test")
	fmt.Printf("curl\n")
	fmt.Printf("%+v\n", err)
	fmt.Printf("%+v\n", curlExec)

	cypressExec, err := execs.Get("cypress/project")
	fmt.Printf("Cypress\n")
	fmt.Printf("%+v\n", err)
	fmt.Printf("%+v\n", cypressExec)

	nonexistingExec, err := execs.Get("non/existing")
	fmt.Printf("Non existing\n")
	fmt.Printf("%+v\n", err)
	fmt.Printf("%+v\n", nonexistingExec)

	// TODO remove resources

	t.Fail()
}
