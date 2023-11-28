import Admonition from "@theme/Admonition";

# Ginkgo

Our dedicated Ginkgo executor allows running Ginkgo tests with Testkube - directly from your Git repository.

* Default command for this executor: `ginkgo`
* Default arguments for this executor command: `-r` `-p` `--randomize-all` `--randomize-suites` `--keep-going` `--trace` `--junit-report` `<reportFile>` `<envVars>` `<runPath>`

Parameters in `<>` are calculated at test execution:

* `<reportFile>` - report file set by `GinkgoJunitReport`, `report.xml` by default
* `<envVars>` - list of environment variables
* `<runPath>` - project path set by `GinkoTestPackage`, location of the test files by default

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

export const ExecutorInfo = () => {
  return (
    <div>
      <Admonition type="info" icon="🎓" title="What is Ginkgo?">
        <ul>
          <li><a href="https://onsi.github.io/ginkgo/">Ginkgo</a> is a popular general purpose testing framework for the Go programming language that, when paired with <a href="https://github.com/onsi/gomega">Gomega</a>, provides a powerful way to write your tests.</li>
          <li>Built on top of Go's testing infrastructure, it lets you write more expressive tests for different use cases: unit tests, integration tests, performance tests, and more.</li>
        </ul>
      </Admonition>
    </div>
  );
}

<ExecutorInfo />

**Check out our [blog post](https://testkube.io/blog/maximize-app-performance-in-kubernetes-with-ginkgo-and-testkube) to learn to write more expressive tests in Go using Ginkgo, Gomega, and Testkube.**

## **Test Environment**

Let's try some simple Ginkgo. Testkube's Ginkgo Executor uses the `ginkgo` binary and allows configuring its behavior using arguments.

Because Ginkgo projects are quite complicated in terms of directory structure, we'll need to load them from a Git directory.

You can find example projects in the repository [here](https://github.com/kubeshop/testkube-executor-ginkgo/tree/main/examples).

Let's create a simple test which will check if an env variable is set to true:

```go
package smoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke", func() {
	Describe("Ginkgo smoke test", func() {
		It("Positive test - should always pass", func(){
			Expect(true).To(Equal(true))
		})
	})
})
```


The default Ginkgo executor: 

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: ginkgo-executor
  namespace: testkube
spec:
  features:
  - artifacts
  - junit-report
  image: kubeshop/testkube-ginkgo-executor:0.0.4
  types:
  - ginkgo/test
```


## **Create a New Ginkgo-based Test**

### Write a Ginkgo Test 

We'll try to check if there are any executors registered on the Testkube demo cluster. To do that we need to check the `/v1/executors`
endpoint. Results should have at least one Executor registered.

```go
package testkube_api_test

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("API Test", func() {
	It("There should be executors registered", func() {
		resp, err := http.Get("https://demo.testkube.io/results/v1/executors")
		Expect(err).To(BeNil())

		executors, err := GetTestkubeExecutors(resp.Body)

		Expect(err).To(BeNil())
		Expect(len(executors)).To(BeNumerically(">", 1))
	})
})

func GetTestkubeExecutors(body io.ReadCloser) ([]testkube.ExecutorDetails, error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	results := []testkube.ExecutorDetails{}
	err = json.Unmarshal(bytes, &results)

	return results, err
}

```

The test is run in the standard Ginkgo bootstrapped project. 
```
go mod init testkube-ginkgo-example
ginkgo bootstrap
```

Everything was pushed to the Git repository.

You can also look at the code in our [examples](https://github.com/kubeshop/testkube-executor-ginkgo/tree/main/examples/testkube-api).

### Add Test to Testkube 

To add a Ginkgo test to Testkube you need to call the `create test` command. We'll assume that our test is in a Git repository.

```bash
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-ginkgo.git --git-path examples/testkube-api --type ginkgo/test --name ginkgo-example-test --git-branch main
```


## **Running a Test**

Let's pass the env variable to our test run:

```bash
 tk run test ginkgo-example-test -f                     

Type:              ginkgo/test
Name:              ginkgo-example-test
Execution ID:      62eceb8df4732077cee099cf
Execution name:    ginkgo-example-test-3
Execution number:  3
Status:            running
Start time:        2022-08-05 10:06:05.467437617 +0000 UTC

... other logs 

Running in parallel across 7 processes
•

Ran 1 of 1 Specs in 0.091 seconds
SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped


Ginkgo ran 1 suite in 7.447676906s
Test Suite Passed

Test execution completed with success in 15.586s 🥇

Watch test execution until complete:
$ kubectl testkube watch execution 62eceb8df4732077cee099cf


Use the following command to get test execution details:
$ kubectl testkube get execution 62eceb8df4732077cee099cf

```

## **Getting Test Results**

We can always get back to the test results: 

```bash
kubectl testkube get execution 62eceb8df4732077cee099cf
```

Output:

```bash
# ....... a lot of Ginkgo logs

ID:         62eceb67f4732077cee099cd
Name        ginkgo-example-test-2
Number:            2
Test name:         ginkgo-example-test
Type:              ginkgo/test
Status:            passed
Start time:        2022-08-05 10:05:27.659 +0000 UTC
End time:          2022-08-05 10:05:43.14 +0000 UTC
Duration:          00:00:15

go: downloading github.com/kubeshop/testkube v1.4.5
go: downloading github.com/onsi/gomega v1.20.0
go: downloading github.com/onsi/ginkgo/v2 v2.1.4
go: downloading github.com/google/go-cmp v0.5.8
go: downloading golang.org/x/net v0.0.0-20220722155237-a158d28d115b
go: downloading gopkg.in/yaml.v3 v3.0.1
go: downloading golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
go: downloading golang.org/x/text v0.3.7
go: downloading go.mongodb.org/mongo-driver v1.10.1
go: downloading github.com/dustinkirkland/golang-petname v0.0.0-20191129215211-8e5a1ed0cff0

Running Suite: TestkubeApi Suite - /tmp/git-sparse-checkout2422275089/repo/examples/testkube-api
================================================================================================
Random Seed: 1659693931 - will randomize all specs

Will run 1 of 1 specs
Running in parallel across 7 processes
•

Ran 1 of 1 Specs in 0.088 seconds
SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped


Ginkgo ran 1 suite in 7.7928584s
Test Suite Passed

Status Test execution completed with success 🥇
```

## **Summary**

Testkube simplifies running Go tests based on Ginkgo and allows them to run in your Kubernetes cluster with ease.
