# What is a Testkube Executor

If tests are written in testing frameworks other than those Testkube supports out-of-the-box, you can write a `custom executor`.

An Executor is a wrapper around a testing framework in the form of a Docker container and run as a Kubernetes Job. Usually, an executor runs a particular test framework binary inside a container. Additionally, it's registered as an Executor Custom Resource in your Kubernetes cluster with a type handler defined (e.g. `postman/collection`).

The Testkube API is responsible for running executions and will pass test data to the executor and parse the results from the execution output.

To create a new script, a user needs to pass `--type`. The API needs it to pair the test type with the executor (executors have handled `types` array defined in CRD), and the API will choose which executor to run based on the handled types.

API will pass `testube.Execution` OpenAPI based document as first argument to binary in executors Docker container,

API assumes that Executor will output JSON data to `STDOUT` and each line is wrapped in `testkube.ExecutorOutput` (like in structured logging idea).


# Custom Executors

## Creating a Custom Executor

You can create a custom executor on your own or by using our executor template (in `go` language):

### Using `testkube-executor-template`

(See the implementation example here: <https://github.com/exu/testkube-executor-example>)

If you are familiar with the `go` programming language, you can use our template repository for new executors:

1. Create new repository from a template [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
2. Clone the newly created repo.
3. Rename the go module from `testkube-executor-template` in the  new name and run `go mod tidy`.

[Testkube](https://github.com/kubeshop/testkube) provides the components to help implement the new runner. 
A `Runner` is a wrapper around a testing framework binary responsible for running tests and parsing tests results. You are not limited to use Testkube's components for the `go` language. Use any language you want - just remember about managing input and output.

Let's try to create new test runner which test if given URI call is successfull (`status code == 200`)

To create new runner we should implement `testkube.Runner` interface first

```go
type Runner interface {
 // Run takes Execution data and returns execution result
 Run(execution testkube.Execution) (result testkube.ExecutionResult, err error)
}
```

As we can see we'll get `Execution` in input - this object is managed by testkube API and will be passed
to your executor. The executor will have information about the execution id and content that should be run on top of your runner. 

An example runner is defined in our template. Using this template will only require implementing the Run method (you can rename `ExampleRunner` to whatever business name best describes your testing framework).

A runner can get data from different sources. We are currently supporting:

- string content (e.g. Postman JSON file)
- URI - content stored on webserver
- Git File - file storeg in Git repo in given path
- Git Dir - whole git repo, or git subdirectory (we'll do spatial checkout to save traffic in case of monorepos)

All possible test definitions are already created and mounted as Kubernetes `Volume` before an executor starts its work. You can get directory path from the `RUNNER_DATADIR` environment variable. 

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

Additionally, if we want to parse test framework test parts (e.g. different test steps), we should make some 
map the particular testing framework and Testkube itself (we have skipped those details here to simplify the example).

If running any testing framework binary, it is good to wrap its output.

Here is an example of [mapping in Testkube Postman Executor](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go#L60), which is using [Postman to Testkube Mapper](https://github.com/kubeshop/testkube-executor-postman/blob/1b95fd85e5b73e9a243fbff59d5e96c27d0f69c5/pkg/runner/newman/mapper.go#L9).


### Deploying your executor

When everything is completed you'll need to build and deploy your runner into Kubernetes cluster.

```sh
docker build -t YOUR_USER/testkube-executor-example:1.0.0 . 
docker push YOUR_USER/testkube-executor-example:1.0.0
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

What we have shown is a most basic executor example. Please visit our internal projects for more examples and the details on implementation:

- [Postman runner implementation](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go)
- [Cypress runner implementation](https://github.com/kubeshop/testkube-executor-cypress/blob/main/pkg/runner/cypress.go)
- [Curl runner implementation](https://github.com/kubeshop/testkube-executor-curl/blob/main/pkg/runner/runner.go)


# Creating executor in a programming langiage other than `go`

([You can find the fully commented code example here](https://github.com/kubeshop/testkube-executor-example-nodejs/blob/main/app.js)).

For go-based executors, we have prepared many handy functions, such as printing valid outputs or wrappers around calling external processes.
Currently, in other languages, you'll need to manage this on your own.

Testkube has simplified test content management. We are supporting several different test content types such as string, uri, git-file and git-dir. The entire  complexity of checking out or downloading test content is covered by Testkube. 

Testkube will store its files and directories in a directory defined by `RUNNER_DATADIR` env and will save the test-content file for:

- string content (e.g., a postman collection is passed as string content read from a JSON file).
- uri (Testkube will get the content of the file defined by the uri).
In the case of git related content: 
- Testkube will checkout the repo content in the current directory.

We have created a simple NodeJS executor (sorry for our Node skills we've tried the best ;) ) 

The executor will get the URI and try to call the HTTP GET method on the passed value, and will return
- success - when status code is 200 
- failed - otherwise 

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

// error will return error info not related to test itself (some issues with executor)
function error(message) {
  console.log(JSON.stringify({ "type": "error", "content": message, })); 
}
  
```

The code is ready and working. With the defaults assumed, `RUNNER_DATADIR` will be `/data` and the file will be saved in the `/data/test-content` directory.

As we can see, we are pushing JSON output to stdin with the console.log function that is based on our [OpenAPI spec - ExecutorOutput](https://kubeshop.github.io/testkube/openapi/).

The two basic output types handled here are:
- in the case of executor failures (non-test related) return `error`, 
- in the case of a test result, return `result` with the test status (success, error)


When the executor code is ready, the next steps are to create:
- A Docker image (create image and push).
- A Kubernetes Executor Custom Resource Definition (CRD).
- The test itself.

We will simplify and use the latest tag here but you should use versioning as a good practice. Currently, Testkube runs the command directly and passes execution information as an argument.

1. Add the runner binary (we have plans to remove this step in a future release):

```sh
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

3. Build and push the docker container (change user/repo to your Docker Hub username): 

```sh
docker build --platform=linux/amd64 -t USER/testkube-executor-example-nodejs:latest -f Dockerfile .
docker push USER/testkube-executor-example-nodejs:latest
```

4. After the image is in place for Kubnernetes to load it, define the executor:

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: example-nodejs-executor
  namespace: testkube
spec:
  executor_type: job
  features: []
  image: kubeshop/testkube-executor-example-nodejs:latest
  types:
    - example/test
```

5. Save the file to a file name, e.g., example-executor.yaml, and apply it into the Kubernetes cluster:

```sh
kubectl apply -f example-executor.yaml
```


When everything is in place, we can add our Testkube tests: 
(Testkube must be installed to add tests. Review the Testkube [installation instructions](/testkube/installing/)).

```
echo "https://httpstat.us/200" | kubectl testkube tests create --name example-test --type example/test
```

As we can see, we need to pass the test name and test type (`example/test` which we defined in our executor CRD). 

Now it's finally time to run our test!

```
kubectl tests run example-test -f

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : example/test
Name          : example-test-string
Execution ID  : 6218ccd2a26fa94ee7a7cfd1
Execution name: moderately-pleasant-labrador


Getting pod logs
Execution completed Got valid status code: 200 OK

.
Use following command to get test execution details:
$ kubectl testkube tests execution 6218ccd2a26fa94ee7a7cfd1



Got valid status code: 200 OK
Test execution completed with sucess in 6.163s 🥇

Use following command to get test execution details:
$ kubectl testkube tests execution 6218ccd2a26fa94ee7a7cfd1


Watch test execution until complete:
$ kubectl testkube tests watch 6218ccd2a26fa94ee7a7cfd1


```

Our test completed successfully! Create another test with a different status code and check to see how it's failing.



# Resources

- [OpenaAPI spec details](https://kubeshop.github.io/testkube/openapi/)
- [Spec in YAML file](https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml)

Go based resources for input and output objects:

- input: [`testkube.Execution`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_execution.go)
- output line: [`testkube.ExecutorOutput`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_executor_output.go)
