---
sidebar_position: 11
sidebar_label: Creating New Test Types
---
# What is a Testkube Executor?

To use a testing framework that is not on the currently supported framework list for Testkube, you can create your custom executor and configure it to run any type of tests that you need. These custom test types can be added to your Testkube installation and/or contributed to our repo. We are very happy to receive executor contributions from our community.

An Executor is a wrapper around a testing framework in the form of a Docker container and run as a Kubernetes job. Usually, an executor runs a particular test framework binary inside a container. Additionally, it is registered as an Executor Custom Resource in your Kubernetes cluster with a type handler defined (e.g. `postman/collection`).

The Testkube API is responsible for running executions and will pass test data to the executor and parse the results from the execution output.

To create a new script, a user needs to pass `--type`. The API uses it to pair the test type with the executor (executors have a handled `types` array defined in CRD), and the API will choose which executor to run based on the handled types.

The API will pass a `testkube.Execution` OpenAPI based document as the first argument to the binary in the executor's Docker container.

The API assumes that the Executor will output JSON data to `STDOUT` and each line is wrapped in `testkube.ExecutorOutput` (as in structured logging).

## **Contribute to the Testkube Project**

We love to improve Testkube with additional features suggested by our users!

Please visit our [Contribution](../9-contributing/contributing.md) page to see the guidelines for contributing to the Testkube project.


# Custom Executors

## **Creating a Custom Executor**

A custom executor can be created on your own or by using our executor template (in `go` language).

### **Using `testkube-executor-template`**

```bash
See the implementation example here: <https://github.com/exu/testkube-executor-example>).
```

If you are familiar with the `go` programming language, use our template repository for new executors:

1. Create a new repository from the template -  [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
2. Clone the newly created repo.
3. Rename the go module from `testkube-executor-template` to the new name and run `go mod tidy`.

[Testkube](https://github.com/kubeshop/testkube) provides the components to help implement the new runner.
A `Runner` is a wrapper around a testing framework binary responsible for running tests and parsing tests results. You are not limited to using Testkube's components for the `go` language. Use any language - just remember about managing input and output.

Let's try to create a new test runner that tests if a given URI call is successful (`status code == 200`).

To create the new runner, we should implement the `testkube.Runner` interface first:

```go
type Runner interface {
 // Run takes Execution data and returns execution result
 Run(execution testkube.Execution) (result testkube.ExecutionResult, err error)
}
```

As we can see, `Execution` is the input - this object is managed by the Testkube API and will be passed to your executor. The executor will have information about the execution id and content that should be run on top of your runner.

An example runner is defined in our template. Using this template will only require implementing the Run method (you can rename `ExampleRunner` to the name that best describes your testing framework).

A runner can get data from different sources. Testkube currently supports:

- String content (e.g. Postman JSON file).
- URI - content stored on the webserver.
- Git File - the file stored in the Git repo in the given path.
- Git Dir - the entire git repo or git subdirectory (Testkube does a spatial checkout to limit traffic in the case of monorepos).

All possible test definitions are already created and mounted as Kubernetes `Volumes` before an executor starts its work. You can get the directory path from the `RUNNER_DATADIR` environment variable.

```go
// TODO: change to a valid name

type ExampleRunner struct {
}

func (r *ExampleRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
 
  // execution.Content could have git repo data
  // We are also passing content files/directories as mounted volume in a directory.

  path := os.Getenv("RUNNER_DATADIR")

  // For example, the Cypress test is stored in the Git repo so Testkube will check it out automatically 
  // and allow you to use it easily.

  uri := execution.Content.Data
  resp, err := http.Get(uri)
  if err != nil {
    return result, err
  }
  defer resp.Body.Close()

  b, err := io.ReadAll(resp.Body)
  if err != nil {
    return result, err
  }

  // If successful, return success result.

  if resp.StatusCode == 200 {
    return testkube.ExecutionResult{
      Status: testkube.ExecutionStatusSuccess,
      Output: string(b),
    }, nil
  }

  // Otherwise, return an error to simplify the example.

  err = fmt.Errorf("invalid status code %d, (uri:%s)", resp.StatusCode, uri)
  return result.Err(err), nil
}

```

A Runner returns `ExecutionResult` or `error` (in the case that the runner can't run the test). `ExecutionResult`
could have different statuses (review the OpenAPI spec for details). In our example, we will focus on `success` and `error`.

Additionally, to parse test framework test parts (e.g. different test steps), create a  
map of the particular testing framework and Testkube itself. Those details have been skipped here to simplify the example.

If running any testing framework binary, it is a best practice to wrap its output.

Here is an example of [mapping in the Testkube Postman Executor](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go#L60), which is using a [Postman to Testkube Mapper](https://github.com/kubeshop/testkube-executor-postman/blob/1b95fd85e5b73e9a243fbff59d5e96c27d0f69c5/pkg/runner/newman/mapper.go#L9).


### **Deploying a Custom Executor**

The following example will build and deploy your runner into a Kubernetes cluster:

```bash
docker build -t YOUR_USER/testkube-executor-example:1.0.0 . 
docker push YOUR_USER/testkube-executor-example:1.0.0
```

When the Docker build completes, register the custom executor using the Testkube cli:

```bash
kubectl testkube create executor --image YOUR_USER/testkube-executor-example:1.0.0 --types "example/test" --name example-executor
```

An example Executor custom resource deployed by Testkube would look the following in yaml:

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: example-executor
  namespace: testkube
spec:
  executor_type: job  
  # 'job' is currently the only type for custom executors
  image: YOUR_USER/testkube-executor-example:1.0.0 
  # pass your repository and tag
  types:
  - example/test      
  # your custom type registered (used when creating and running your testkube tests)
  content_types:
  - string                               # test content as string 
  - file-uri                             # http based file content
  - git-file                             # file stored in Git
  - git-dir                              # entire dir/project stored in Git
  features: 
  - artifacts                            # executor can have artifacts after test run (e.g. videos, screenshots)
  - junit-report                         # executor can have junit xml based results
  meta:
   iconURI: http://mydomain.com/icon.jpg # URI to executor icon
   docsURI: http://mydomain.com/docs     # URI to executor docs
   tooltips:
    name: please enter executor name     # tooltip for executor fields
```

Finally, create and run your custom tests by passing `URI` as the test content:

```bash
# create 
echo "http://google.pl" | kubectl testkube create test --name example-google-test --type example/test 
# and run it in testkube
kubectl testkube run test example-google-test
```

This is a very basic example of a custom executor. Please visit our internal projects for more examples and the details on implementation:

- [Postman runner implementation](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go).
- [Cypress runner implementation](https://github.com/kubeshop/testkube-executor-cypress/blob/main/pkg/runner/cypress.go).
- [Curl runner implementation](https://github.com/kubeshop/testkube-executor-curl/blob/main/pkg/runner/runner.go).


## **Creating a Custom Executor in a Programming Language other than `Go`**

[You can find the fully commented code example here](https://github.com/kubeshop/testkube-executor-example-nodejs/blob/main/app.js).

For Go-based executors, we have prepared many handy functions, such as printing valid outputs or wrappers around calling external processes.
Currently, in other languages, you'll need to manage this on your own.

Testkube has simplified test content management. We are supporting several different test content types such as string, uri, git-file and git-dir. The entire complexity of checking out or downloading test content is covered by Testkube.

Testkube will store its files and directories in a directory defined by the `RUNNER_DATADIR` env and will save the test-content file for:

- String content (e.g., a postman collection is passed as string content read from a JSON file).
- URI - Testkube will get the content of the file defined by the uri.
- Git related content - Testkube will checkout the repo content in the current directory.

To be able to proceed with this guide, Testkube should be installed. Review the Testkube [installation instructions](../1-installing.md).

We have created a simple NodeJS executor.

The executor will get the URI and try to call the HTTP GET method on the passed value, and will return:

- **success** - when the status code is 200. 
- **failed** - for any other status code.

```javascript
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");

const args = process.argv.slice(2);
if (args.length == 0) {
  error("Please pass arguments");
  process.exit(1);
}

var uri;
if (!process.env.RUNNER_DATADIR) {
  error("No valid data directory detected");
  process.exit(1);
}

const testContentPath = path.join(process.env.RUNNER_DATADIR, "test-content");
uri = fs.readFileSync(testContentPath, { encoding: "utf8", flag: "r"});

https.get(uri, (res) => {
    if (res.statusCode == 200) {
      successResult("Got valid status code: 200 OK");
    } else {
      errorResult("Got invalid status code");
    }
  })
  .on("error", (err) => { error("Error: " + err.message); });


function errorResult(message) {
  console.log(JSON.stringify({ "type": "result", "result": { "status": "error", "errorMessage": message, }}));
}

function successResult(output) {
  console.log(JSON.stringify({ "type": "result", "result": { "status": "success", "output": output, }}));
}

// 'error' will return the error info not related to the test itself (issues with the executor)
function error(message) {
  console.log(JSON.stringify({ "type": "error", "content": message, })); 
}
  
```

The code is ready and working. With the defaults assumed, `RUNNER_DATADIR` will be `/data` and the file will be saved in the `/data/test-content` directory.

As we can see, we are pushing JSON output to `stdin` with the `console.log` function that is based on our [OpenAPI spec - ExecutorOutput](https://kubeshop.github.io/testkube/openapi/).

The two basic output types handled here are:

- For executor failures (non-test related), return `error`. 
- For a test result, return `result` with the test status (success, error).


When the executor code is ready, the next steps are to create:

- A Docker image (create image and push).
- A Kubernetes Executor Custom Resource Definition (CRD).
- The test itself.

We will simplify and use the latest tag for the steps below but a best practice is to use versioning. Currently, Testkube runs the command directly and passes the execution information as an argument.

1. Add the runner binary (we have plans to remove this step in a future release):
```bash
#!/usr/bin/env sh
node app.js "$@"
```

2. Add the runner binary into the Dockerfile:
```Dockerfile
FROM node:17

# Create app directory
WORKDIR /usr/src/app

# Bundle app source
COPY runner /bin/runner
RUN chmod +x /bin/runner
COPY app.js app.js

EXPOSE 8080
CMD [ "/bin/runner" ]
```

3. Build and push the docker container (change `user/repo` to your Docker Hub username): 
```bash
docker build --platform=linux/amd64 -t USER/testkube-executor-example-nodejs:latest -f Dockerfile .
docker push USER/testkube-executor-example-nodejs:latest
```

4. After the image is in place for Kubernetes to load, create the executor:
```bash
kubectl testkube create executor --image kubeshop/testkube-executor-example-nodejs:latest --types "example/test" --name example-nodejs-executor
```


When everything is in place, we can add our Testkube tests.

```
echo "https://httpstat.us/200" | kubectl testkube create test --name example-test --type example/test
```

As we can see, we need to pass the test name and test type `example/test` (which we defined in our executor CRD).

Now it's finally time to run our test!

```
kubectl tests run example-test -f

Type          : example/test
Name          : example-test-string
Execution ID  : 6218ccd2a26fa94ee7a7cfd1
Execution name: moderately-pleasant-labrador


Getting pod logs
Execution completed Got valid status code: 200 OK

.
Use the following command to get test execution details:
$ kubectl testkube get execution 6218ccd2a26fa94ee7a7cfd1



Got valid status code: 200 OK
Test execution completed with success in 6.163s 🥇

Use the following command to get test execution details:
$ kubectl testkube get execution 6218ccd2a26fa94ee7a7cfd1


Watch the test execution until complete:
$ kubectl testkube watch execution 6218ccd2a26fa94ee7a7cfd1


```

Our test completed successfully! Create another test with a different status code and check to see the result.



## **Resources**

- [OpenAPI spec details](https://kubeshop.github.io/testkube/openapi/).
- [Spec in YAML file](https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml).

Go-based resources for input and output objects:

- Input: [`testkube.Execution`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_execution.go)
- Output line: [`testkube.ExecutorOutput`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_executor_output.go)
