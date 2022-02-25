# Creating your own Executor

Executors are programs which are encapsulated in docker containers.

Only one restriction is that they need to implement communication interface:

- INPUT : get JSON (`testube.Execution`) OpenAPI based document,
- OUTPUT : stream of json output lines (`testkube.ExecutorOutput`) - each output line is simply wrapped in this JSON, like in structured logging idea.

In order to be able to run tests using some new tools for which there is no executor, it is possible to create a custom executor from the [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).

## Steps for creating executor

You can check full example implementation here: <https://github.com/exu/testkube-executor-example>

### Setup repository

- Create new rpository from template [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
- Clone the newly created repo.
- Rename the go module from `testkube-executor-template` in whole project to the new name & run `go mod tidy`.

### Implement Runner Components

[Testkube](https://github.com/kubeshop/testkube) provides the components to help implement a new runner which is responsible for running and parsing results. But you're not limited to use our components for `go` language - you can you whatever language you want - just remember about managing input and output.

Let's try to create new test runner which test if given URI call is successfull (`status code == 200`)

To create new runner we should implement `testkube.Runner` interface first

```go
type Runner interface {
 // Run takes Execution data and returns execution result
 Run(execution testkube.Execution) (result testkube.ExecutionResult, err error)
}
```

As we can see we'll get `Execution` in input - this object is managed by testkube API and will be passed
to your executor - it'll have information about execution id and content which should be run on top of your runner. Example runner is defined in our template - so if you'll use it only thing which need to be done is implementing Run method (you can rename ExampleRunner to whatever you want)

```go
// ExampleRunner for template - change me to some valid runner
type ExampleRunner struct {
}

func (r *ExampleRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
 return testkube.ExecutionResult{
  Status: testkube.StatusPtr(testkube.SUCCESS_ExecutionStatus),
  Output: "exmaple test output",
 }, nil
}
```

Runner need to return `ExecutionResult` or `error` (in case of runner can't run tests), ExecutionResult
could have different statuses (look at OpenAPI spec for details) - we'll focus on `success` and `error`

Let's assume that user will create test which content will be simply URI to test.

```go
func (r *CurlRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {

}
```

### Deploying your executor

When everything is completed you'll need to build and deploy your runner into Kubernetes cluster.

```sh
docker build -t YOUR_USER/testkube-executor-example . 
docker push YOUR_USER/testkube-executor-example
```

When docker containers are finally here we're ready to register our executor:
Create yaml file with definition: (`executor.yaml`)

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: example-executor
  namespace: testkube
spec:
  executor_type: job
  image: kubeshop/testkube-example-executor:0.0.1 # pass your repository and tag
  types:
  - example/test
```

and apply it on your cluster:

```sh
kubectl apply -f executor.yaml
```

Now we're ready to create and run your custom tests by passing URI as test content

```sh
# create 
echo "http://google.pl" | kubectl testkube tests create --name example-google-test --type example/test 
# and run it in testkube
kubectl testkube tests run example-google-test
```

That's all for the most basic executor example, you can look our internal projects for more examples
and details how it's implemented:

## Resources


- [OpenaAPI spec details](https://kubeshop.github.io/testkube/openapi/)
- [Spec in YAML file](https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml)

Go based resources for input and output objects:

- input: [`testkube.Execution`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_execution.go)
- output line: [`testkube.ExecutorOutput`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_executor_output.go)
