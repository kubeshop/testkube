# What is Testkube Executor

When your tests are written in other testing frameworks than Testkube supports out-of-the-box, you can write `custom executor`.

Executor is wrapper around testing framework in form of *Docker container* run as *Kubernetes Job*. Usually it'll run particular test framework binary inside container. Additionally it's registered as `Executor` Custom Resource in your Kubernetes cluster with type handler defined (e.g. `postman/collection`).

Testkube API is responsible for running executions, it'll pass test data to executor and parse results from execution output. 

To create new script user need to pass `--type` - API need it to pair script type with executor (executor have handled `types` array defined in CRD), and API will choose which executor to run based on handled types.

API will pass `testube.Execution` OpenAPI based document as first argument to binary in executors Docker container,

API assume that Executor will output JSON data to `STDOUT` and each line is wrapped in `testkube.ExecutorOutput` (like in structured logging idea).


# Custom Executors

## Creating custom executor

You can create custom executor by your own, or by using our executor template (in `go` language):

### Using `testkube-executor-template`

(You can check full example implementation here: <https://github.com/exu/testkube-executor-example>)

If you're familiar with `go` programming language you can use our template repository for new executors:

1. Create new repository from template [testkube-executor-template](https://github.com/kubeshop/testkube-executor-template).
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

all possible test definitions are already created and mounted as Kubernetes `Volume` before executor starts its work. You can get directory path from `RUNNER_DATADIR` environment variable. 

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

Additionally we could want to parse test framework test parts (e.g. different test steps) we should make some 
mapping between particular testing framework and Testkube itself (keep in mind that we've skipped those details here to simplify example).

If we're running any testing framework binary it's good to wrap its output 

Example of [mapping in Testkube Postman Executor](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go#L60), which using [Postman to Testkube Mapper](https://github.com/kubeshop/testkube-executor-postman/blob/1b95fd85e5b73e9a243fbff59d5e96c27d0f69c5/pkg/runner/newman/mapper.go#L9)



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

That's all for the most basic executor example, you can look our internal projects for more examples and details how it's implemented:

- [Postman runner implementation](https://github.com/kubeshop/testkube-executor-postman/blob/main/pkg/runner/newman/newman.go)
- [Cypress runner implementation](https://github.com/kubeshop/testkube-executor-cypress/blob/main/pkg/runner/cypress.go)
- [Curl runner implementation](https://github.com/kubeshop/testkube-executor-curl/blob/main/pkg/runner/runner.go)


# Creating executor in other programming language (than `go`)

([You can find full commented code example here](https://github.com/kubeshop/testkube-executor-example-nodejs/blob/main/app.js))

For go-based executors we've prepared a lot of handy functions (like printing valid outputs or wrappers around calling external processes)
In other languages (for now) you'll need to manage this by your own. 

One thing which Testkube simplified is test content management. As we're supporting several different test content types (like string,uri,git-file,git-dir)
The whole complexity of checking out or downloading is covered by Testkube. 

Testkube will store it's files and directories in directory defined by `RUNNER_DATADIR` env 
And will save `test-content` file for:
- string content (e.g. postman collection is passed as string content read from json file)
- uri (testkube will get content of file defined by uri)
In case of git related content: 
- testkube will checkout repo content in that directory

We've created simple NodeJS executor (sorry for our Node skills we've tried the best ;) ) 

Executor will get URI and try to call HTTP GET method on passed value, and will return 
- success - when status code will be 200 
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

Code is ready and working - we're assuming defaults so `RUNNER_DATADIR` will be `/data` and our file will be in `/data/test-content` file.

As we can see we're pushing JSON output to stdin with console.log function (it's based on our [OpenAPI spec - ExecutorOutput](https://kubeshop.github.io/testkube/openapi/))

Two basic output types are handled here:
- in case of executor failures (non-test related) we should return `error`, 
- in case of test result we should return `result` with test status (success, error)


Now when executor code is ready we need additional steps:
- Docker image (create image and push)
- Kubernetes Executor Custom Resource definition
- Create test

Let's start with Docker: 
We'll simplify and use latest tag here - but you should use versioning as good practice.
As for now testkube runs directly command and is passing execution information as argument 
We need to add runner binary (but we have plans to remove need of this step)

```sh
#!/usr/bin/env sh
node app.js "$@"
```

And add it into our Dockerfile

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

Now let's build and push our docker container (change user/repo to your Docker Hub username): 
```sh
docker build --platform=linux/amd64 -t USER/testkube-executor-example-nodejs:latest -f Dockerfile .
docker push USER/testkube-executor-example-nodejs:latest
```

After we have our image in place where Kubnernetes can load it we need to define our executor: 

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

Save it somewhere e.g. `example-executor.yaml` and apply into Kubernetes cluster: 

```sh
kubectl apply -f example-executor.yaml
```


When everything in place we can now start adding our Testkube tests 
(We'll need testkube for this so head to [installation instructions](/testkube/installing/))

```
echo "https://httpstat.us/200" | kubectl testkube tests create --name example-test --type example/test
```

As we can see we need to pass test name and test type (`example/test` which we defined in our executor CRD). 

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

Yay our test completes with success!
(you can try to create another test with diferent status code and check how it's failing)



# Resources

- [OpenaAPI spec details](https://kubeshop.github.io/testkube/openapi/)
- [Spec in YAML file](https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml)

Go based resources for input and output objects:

- input: [`testkube.Execution`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_execution.go)
- output line: [`testkube.ExecutorOutput`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_executor_output.go)
