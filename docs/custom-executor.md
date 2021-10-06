# Creating your own Executor

In order to be able to run tests using some new tools for which there is no executor, there is possibility to create a custom executor from the [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).

## Steps for creating executor

### Setup repository

- Create new rpository from template [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).
- Clone the newly created repo.
- Rename the go module from `kubetest-executor-template` in whole project to the new name & run `go mod tidy`.

### Implement Runner Components

[Kubtest](https://github.com/kubeshop/kubtest) provides the components to help implement a new executor and only the explicit runner needs to be implemented but you're not forced to use it - just implement OpenAPI Spec for executor 
and you're good.
In this example for sake of simplicity we'll use `kubtest` components to implement executor they help us with:
- Defining basic HTTP Rest based server skeleton
- Creating job queue - our tests will be run in async mode
- Running runners - this is the only part which need to be implmented when using `kubtest` components.


To implement new executor we should do following: 

- Define the input format for tests.
  In order to communicate effectively with executor we need to define a format on how to structure the tests. And bellow is an example for the curl based tests.

```json
{
    "command": ["curl",
        "https://reqbin.com/echo/get/json",
        "-H",
        "'Accept: application/json'"
    ],
    "expected_status": 200,
    "expected_body": "{\"success\":\"true\"}"
}
```

The output will be stored in the `content` field of the request body and the request body will look like:

```json
{
    "type": "curl/test",
    "name": "some-custom-execution-name",
    "content": "{\"command\": [\"curl\", \"https://reqbin.com/echo/get/json\", \"-H\", \"'Accept: application/json'\"],\"expected_status\":200,\"expected_body\":\"{\\\"success\\\":\\\"true\\\"}\"}"
}
```

There is also the field `params` field in the request that can be used to pass some aditional key value pairs to the runner. You need to implement them by your own - they are often used in runners to pass additional variables to test.

- Create execution storage repository.
  There is a storage repository `result.MongoRepository` provided implemented using mongo DB but there is the possibility to provide a new storage type.
  The repository needs to implement the interface bellow, it can use inmemory storage,a database or whatever fits the needs. This is only used for execution scheduling, the final results will be centralized by the api. You can also use 
  built-in repository (for now we're handling mongo repository only)

```go
type Repository interface {
    // Get gets execution result by id
    Get(ctx context.Context, id string) (kubtest.Execution, error)
    // Insert inserts new execution result
    Insert(ctx context.Context, result kubtest.Execution) error
    // Update updates execution result
    Update(ctx context.Context, result kubtest.Execution) error
    // QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
    QueuePull(ctx context.Context) (kubtest.Execution, error)
}
```

- Prepare docker for the type of the executor.
  In this step the docker should be configured to make sure that the runner has all dependencies installed and ready to use. 
  In the case of the [kubtest-executor-curl](https://github.com/kubeshop/kubtest-executor-curl) only installing curl was needed.

```docker
FROM golang:1.17
WORKDIR /build
COPY . .
ENV GONOSUMDB=github.com/kubeshop/* 
ENV CGO_ENABLED=0 
ENV GOOS=linux
RUN cd cmd/executor;go build -o /app -mod mod -a .

FROM alpine
RUN apk --no-cache add ca-certificates && \
    apk --no-cache add curl
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8083
ENTRYPOINT ["/bin/app"]
```

- Create new runner.
  Runner should contain the logic to run the test and to verify the expectations based on the interface from bellow.

```go
// Runner interface to abstract runners implementations
type Runner interface {
    // Run takes Execution data and returns execution result
    Run(execution kubtest.Execution) kubtest.ExecutionResult
}
```

  For the curl executor provide the struct that matches the exact structure of the test input format which runner will take as the input(described above).

```go
type CurlRunnerInput struct {
    Command        []string `json:"command"`
    ExpectedStatus int      `json:"expected_status"`
    ExpectedBody   string   `json:"expected_body"`
}
```

  And bellow is the business logic for the curl executor and it executes the curl command given as input, takes the output, tests the expectations and returns the result.

```go
func (r *CurlRunner) Run(execution kubtest.Execution) kubtest.ExecutionResult {
    var runnerInput CurlRunnerInput
    err := json.Unmarshal([]byte(execution.ScriptContent), &runnerInput)
    if err != nil {
        return kubtest.ExecutionResult{
            Status: kubtest.ExecutionStatusError,
        }
    }
    command := runnerInput.Command[0]
    runnerInput.Command[0] = CurlAdditionalFlags
    output, err := process.Execute(command, runnerInput.Command...)
    if err != nil {
        r.Log.Errorf("Error occured when running a command %s", err)
        return kubtest.ExecutionResult{
            Status:       kubtest.ExecutionStatusError,
            ErrorMessage: fmt.Sprintf("Error occured when running a command %s", err),
        }
    }

    outputString := string(output)
    responseStatus := getResponseCode(outputString)
    if responseStatus != runnerInput.ExpectedStatus {
        return kubtest.ExecutionResult{
            Status:       kubtest.ExecutionStatusError,
            RawOutput:    outputString,
            ErrorMessage: fmt.Sprintf("Response statut don't match expected %d got %d", runnerInput.ExpectedStatus, responseStatus),
        }
    }

    if !strings.Contains(outputString, runnerInput.ExpectedBody) {
        return kubtest.ExecutionResult{
            Status:       kubtest.ExecutionStatusError,
            RawOutput:    outputString,
            ErrorMessage: fmt.Sprintf("Response doesn't contain body: %s", runnerInput.ExpectedBody),
        }
    }

    return kubtest.ExecutionResult{
        Status:    kubtest.ExecutionStatusSuceess,
        RawOutput: outputString,
    }
}
```

### Deploying your executor

When everything is completed you'll need to build and deploy your runner into Kubernetes cluster. 

```
docker build -t YOUR_USER/kubtest-executor-curl . 
docker push YOUR_USER/kubtest-executor-curl
```

and define some deployment: 
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: curl-executor-deployment
  labels:
    app: curl-executor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: curl-executor
  template:
    metadata:
      labels:
        app: curl-executor
    spec:
      containers:
      - name: executor
        image: YOUR_USER/kubtest-executor-curl:latest
        env:
        - name: EXECUTOR_PORT
          value: "8083"
        ports:
        - containerPort: 8083
        resources:
          limits:
            cpu: 300m
            ephemeral-storage: 1Gi
            memory: 200Mi
          requests:
            cpu: 100m
            ephemeral-storage: 1Gi
            memory: 100Mi
```

(You should tune deployment for production use)

### Add Executor to Kubtest

Last thing which needs to be done is to create binding for your executor to some type 
We've defined `curl/test` type above so let's bind this type so kubtest would be aware of it. 
To do this we need to create new Executor Custom Resource

```
apiVersion: executor.kubtest.io/v1
kind: Executor
metadata:
  annotations:
    meta.helm.sh/release-name: kubtest
    meta.helm.sh/release-namespace: default
  name: curl-executor
spec:
  executor_type: rest
  types:
  - curl/test
  uri: http://kubtest-curl-executor:8086
```

