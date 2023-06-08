//go:build e2e

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/kubeshop/testkube/test/e2e/testkube"
)

var namespace = "testkube"

func init() {
	if ns, ok := os.LookupEnv("NAMESPACE"); ok {
		namespace = ns
	}

}

var install = flag.Bool("install", false, "test")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
func TestE2E(t *testing.T) {
	a := require.New(t)
	test := testkube.NewTestkube(namespace)
	testName := fmt.Sprintf("test-%s", rand.Name())
	collectionFile := "test.postman_collection.json"

	t.Logf("Sctipt name: %s", testName)
	t.Logf("Collection file name: %s", collectionFile)
	t.Logf("Kubernetes namespace: %s", namespace)

	t.Run("install test", func(t *testing.T) {
		if !*install {
			t.Skip("install flag not passed ignoring install test")
		}
		// given
		test.Output = "json"

		// uninstall first before installing
		test.Uninstall()

		// TODO change to watch
		sleep(t, 10*time.Second)

		// when
		out, err := test.Install()

		// then
		a.NoError(err)
		a.Contains(string(out), "STATUS: deployed")
		a.Contains(string(out), "Visit http://127.0.0.1:8088 to use your application")

		// TODO change to watch for changes
		sleep(t, time.Minute)
	})

	t.Run("tests management", func(t *testing.T) {
		// given
		out, err := test.CreateTest(testName, collectionFile)
		a.NoError(err)
		a.Contains(string(out), "Test created")

		// when
		out, err = test.List()
		a.NoError(err)

		// then
		a.Contains(string(out), testName)

		sleep(t, 5*time.Second)
	})

	t.Run("tests run", func(t *testing.T) {
		// given
		executionName := rand.Name()

		// when
		out, err := test.StartTest(testName, executionName)
		a.NoError(err)

		// then check if info about collection steps exists somewhere in output
		a.Contains(string(out), "Kasia.in Homepage")
		a.Contains(string(out), "Google")

		// then check if tests completed with success
		a.Contains(string(out), "Test execution completed with success")

		executionID := GetExecutionID(out)
		t.Logf("Execution completed ID: %s", executionID)
		a.NotEmpty(executionID)

		out, err = test.Execution(testName, executionID)
		// check tests results for postman collection
		a.Contains(string(out), "Google")
		a.Contains(string(out), "Successful GET request")
		// check tests results for postman collection
		a.Contains(string(out), "Kasia.in Homepage")
		a.Contains(string(out), "Body matches string")
	})

	t.Run("delete test", func(t *testing.T) {
		// given
		out, err := test.DeleteTest(testName)
		a.NoError(err)
		a.Contains(string(out), "Succesfully deleted")

		// when
		out, err = test.List()
		a.NoError(err)

		// then
		a.NotContains(string(out), testName)
	})

	sleep(t, time.Second)

	// t.Run("cleaning helm release", func(t *testing.T) {
	// 	out, err := test.Uninstall()
	// 	a.NoError(err)
	// 	a.Contains(string(out), "uninstalled")
	// })

}

func sleep(t *testing.T, d time.Duration) {
	t.Logf("Waiting for changes for %s (because I can't watch yet :P)", d)
	time.Sleep(d)
}

func GetExecutionID(out []byte) string {
	r := regexp.MustCompile("kubectl testkube get execution test ([0-9a-zA-Z]+)")
	matches := r.FindStringSubmatch(string(out))
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
