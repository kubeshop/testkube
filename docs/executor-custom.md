# What is Testkube Executor

When your tests are written in other testing frameworks than Testkube supports out-of-the-box, you can write `custom executor`.

Executor is wrapper around testing framework in form of *Docker container* run as *Kubernetes Job*. Usually it'll run particular test framework binary inside container. Additionally it's registered as `Executor` Custom Resource in your Kubernetes cluster with type handler defined (e.g. `postman/collection`).

Testkube API is responsible for running executions, it'll pass test data to executor and get parse results from eecutor output. 

To create new script user need to pass `--type` - API need it to pair script type with executor (executor have handled `types` array defined in CRD), and API will choose which executor to run based on handled types.

API will pass `testube.Execution` OpenAPI based document as first argument to binary in executors Docker container,

API assume that Executor will output data to `STDOUT` and each line is wrapped in `testkube.ExecutorOutput` (like in structured logging idea).


# Creating your own Executor

In order to be able to run tests using some new tools for which there is no executor, it is possible to create a **custom executor** from the [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).

## Steps for creating executor

You can check full example implementation here: <https://github.com/exu/testkube-executor-example>

### Setup repository

- Create new repository from template [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
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
  Status: testkube.ExecutionStatusSuccess,
  Output: "exmaple test output",
 }, nil
}
```

Runner need to return `ExecutionResult` or `error` (in case of runner can't run tests), ExecutionResult
could have different statuses (look at OpenAPI spec for details) - we'll focus on `success` and `error`

Additionally we could want to parse test framework test parts (e.g. different test steps) we should make some 
mapping between particular testing framework and Testkube itself (keep in mind that we've skipped those details here to simplify example).

Example of [mapping in Testkube Postman Executor](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go#L60), which using [Postman to Testkube Mapper](https://github.com/kubeshop/testkube-executor-postman/blob/1b95fd85e5b73e9a243fbff59d5e96c27d0f69c5/pkg/runner/newman/mapper.go#L9)



### Deploying your executor

When everything is completed you'll need to build and deploy your runner into Kubernetes cluster.

```sh
docker build -t YOUR_USER/testkube-executor-example:1.0.0 . 
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
  executor_type: job   # job is the only one for now for custom executors
  image: YOUR_USER/testkube-executor-example:1.0.0 # pass your repository and tag
  types:
  - example/test       # your custom type registered (used when creating and running your testkube tests)
  contentTypes:
	- string             # test content as string 
	- file-uri           # http based file content
	- git-file           # file stored in Git
	- git-dir            # whole dir/project stored in Git
  features: 
	- artifacts          # executor can have artifacts after test run (e.g. videos, screenshots)
	- junit-report       # executor can have junit xml based results

# remove any contentTypes and features which will be not implemented by your executor

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


# Custom Executors



## Creating custom executor

You can create custom executor by your own, or by using our executor template (in `go` language):

### Using `testkube-executor-template`

(You can check full example implementation here: <https://github.com/exu/testkube-executor-example>)

If you're familiar with `go` programming language you can use our template repository for new executors:

1. Create new rpository from template [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
2. Clone the newly created repo.
3. Rename the go module from `testkube-executor-template` in whole project to the new name & run `go mod tidy`.

[Testkube](https://github.com/kubeshop/testkube) provides the components to help implement the new runner. 
`Runner` is a wrapper around testing framework binary responsible for running tests and parsing tests results. But you're not limited to use our components for `go` language - you can you whatever language you want - just remember about managing input and output.

Let's try to create new test runner which test if given URI call is successfull (`status code == 200`)

To create new runner we should implement `testkube.Runner` interface first

```go
type Runner interface {
 // Run takes Execution data and returns execution result
 Run(execution testkube.Execution) (result testkube.ExecutionResult, err error)
}
```

As we can see we'll get `Execution` in input - this object is managed by testkube API and will be passed
to your executor - it'll have information about execution id and content which should be run on top of your runner. 

Example runner is defined in our template - so if you'll use it only thing which need to be done is implementing Run method (you can rename `ExampleRunner` to whatever business name describing your testing framework)

Runner can get data from different sources - for now we're supporting:

- string content (e.g. Postman JSON file)
- URI - content stored on webserver
- Git File - file storeg in Git repo in given path
- Git Dir - whole git repo, or git subdirectory (we'll do spatial checkout to save traffic in case of monorepos)

```go
// TODO: change me to some valid name
type ExampleRunner struct {
}

func (r *ExampleRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
 
  // execution.Content could have git repo data
  // we're also passing content files/directories as mounted volume in directory
  path := os.Getenv("RUNNER_DATADIR")

  // e.g. Cypress test is stored in Git repo so Testkube will checkout it automatically 
  // and allow you to use it easily


 
	uri := execution.Content.Data
	resp, err := http.Get(uri)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	// if get is successful return success result
	if resp.StatusCode == 200 {
		return testkube.ExecutionResult{
			Status: testkube.ExecutionStatusSuccess,
			Output: string(b),
		}, nil
	}

	// else we'll return error to simplify example
	err = fmt.Errorf("invalid status code %d, (uri:%s)", resp.StatusCode, uri)
	return result.Err(err), nil
}
```

Runner need to return `ExecutionResult` or `error` (in case of runner can't run tests), ExecutionResult
could have different statuses (look at OpenAPI spec for details) - we'll focus on `success` and `error`

Let's assume that user will create test which content will be simply URI to test.

```go
func (r *CurlRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {

}
```

### Using custom language

## Deploying your executor

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

Now we're ready to create and run your custom tests by passing URI as test content (keep in mind that in our example we're using simple string content stored in `Content.Data` string)

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
