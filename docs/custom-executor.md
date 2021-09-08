# Creating your own Executor

In order to be able to run tests using some new tools for which there is no executor, there is possibility to create a custom executor from the [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).

## Steps for creating executor

### Setup repository

- Fork from [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).
- Clone the newly created repo.
- Rename the go module from kubetest-executor-template to the new name & run `go mod tidy`.

### Implement Runner Components

[Kubtest](https://github.com/kubeshop/kubtest) provides the components to help implement a new executor and only the explicit runner needs to be implemented.

- Define an input format for the tests.
  In order to communicate effectivele with executor we need to define a format on how to structure the tests. And bellow is an example for the curl based tests.

    ```js
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

    It will be stored in the `content` field of the request body and the request body will look like:

    ```js
    {
        "type": "curl",
        "name": "test1",
        "content": "{\"command\": [\"curl\", \"https://reqbin.com/echo/get/json\", \"-H\", \"'Accept: application/json'\"],\"expected_status\":200,\"expected_body\":\"{\\\"success\\\":\\\"true\\\"}\"}"
    }
    ```

    There is also the field `params` field in the request that can be used to pass some aditional key value pairs to the runner.

- Create execution storage repository.
  There is a storage repository `result.MongoRepository` provided implemented using mongo DB but there is the possibility to provide a new storage type.
  The repository needs to implement the interface bellow, it can use inmemory storage,a database or whatever fits the needs. This is only used for execution scheduling, the final results will be centralized by the api.

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
  In the case of the [kubtest-executor-curl-example](https://github.com/kubeshop/kubtest-executor-curl-example) only installing curl was needed.

    ```docker
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
