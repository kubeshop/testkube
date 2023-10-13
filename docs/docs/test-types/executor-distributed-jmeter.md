# Distributed JMeter Executor

The Distributed JMeter Executor is an extension of JMeter Executor which can run the JMeter Tests in distributed mode by creating slave pods and distributing the test among them.

## What is an Executor?

An Executor is nothing more than a program wrapped into a Docker container which gets a JSON (testube.Execution) OpenAPI based document as an input and returns a stream of JSON output lines (testkube.ExecutorOutput), where each output line is simply wrapped in this JSON, similar to the structured logging idea.

## Features

This executor is an extension of the JMeter executor and has all the features of the JMeter executor. In addition to that, it has the following features:

- Can run JMeter tests in distributed mode by creating slave pods and distributing the test among them.
- Supports defining plugins for a test in a Git repo by placing plugins in a directory named `plugins` in the test folder.
- Supports overriding the JMeter `user.properties` file by placing a custom `user.properties` file in the test folder.

## Usage

### Supported Environment Variables

1. **MASTER_OVERRIDE_JVM_ARGS/SLAVES_OVERRIDE_JVM_ARGS**: Used to override default memory options for JMeter master/slaves. Example: `MASTER_OVERRIDE_JVM_ARGS=-Xmn256m -Xms512m -Xmx512m`.

2. **SLAVES_COUNT**: Specifies the number of slave pods required for Distributed JMeter tests. Example: `SLAVES_COUNT=3`. Default value of `SLAVES_COUNT` is 1.

3. **MASTER_ADDITIONAL_JVM_ARGS/SLAVES_ADDITIONAL_JMETER_ARGS**: Allows exporting additional JVM arguments for slaves/master. Example: `MASTER_ADDITIONAL_JVM_ARGS=-Xmx1024m -Xms512m -XX:MaxMetaspaceSize=256m`.

4. **SLAVES_ADDITIONAL_JMETER_ARGS**: Provides additional JVM arguments for JMeter server/slaves. Example: `SLAVES_ADDITIONAL_JMETER_ARGS=jmeter-server -Jserver.rmi.ssl.disable=true -Dserver_port=60000`.

### Guide
The guide below will provide you the details about how to run a Jmeter test in a distributed environment.

1. File option:
   When you provide a test (.jmx) file to `Distributed JMeter (JMeter in distributed mode)`, the executor of `Distributed JMeter` will spawn number of slaves pods specified by user through the `SLAVES_COUNT` environment variable as described above and run the test on all the slave pods.  

2. Git Option: 
   Using Git flow of the executor, we can use advanced features of the `Distributed JMeter` executor which is not possible with the JMeter executor:

    - Additional files required by a particular test like a CSV or JSON file can be provided through Git repo. There is an example of using a CSV file by the test (.jmx) file in the `example` folder of `Distributed JMeter`.
    - Dynamic plugins are required for a test by keeping the plugins inside the test folder in a directory named `plugins` in the Git repo.
    - Overriding the JMeter `user.properties` can be provided by using custom `user.properties` file in the Git repo.

To use the Git option and to take advantage of all the above features, the user should have the following directory structure in the Git repo:

```
   github.com/`<username>/<reponame>`/---

                                       |-test1/---
                                                |- testfile1.jmx
                                                |- userdata.csv ( or any other additional file )
                                                |- user.properties
                                                |- plugins/---
                                                            |- plugin-manager.jar
                                                            |- <jar file of any other required plugins to run test1>

                                       |-test2/---
                                                |- testfile2.jmx
                                                |- userdata.json ( or any other additional file )
                                                |- user.properties
                                                |- plugins/---
                                                            |- plugin-manager.jar
                                                            |- <jar file of any other required plugins to run test2>
 ```                                                          
                                                
For additional info, see the [GitFlow Example test for Distributed JMeter](https://github.com/kubeshop/testkube/blob/develop/contrib/executor/jmeterd/examples/gitflow/README.md).

## Prerequisites

Make sure the following tools are installed on your machine and available in your PATH:

- [JMeter](https://jmeter.apache.org/download_jmeter.cgi) - A pure Java application designed to load test functional behavior and measure performance.

### Setup

1. Create a directory called `data/` where JMeter will run and store results (the best practice is to create it in the project root because it is git-ignored).
2. Create a JMeter XML project file and save it as a file named `test-content` in the newly created `data/` directory.
3. Create an execution JSON file and save it as a file named `execution.json` based on the template below (the best practice is to save it in the temp/ folder in the project root because it is git-ignored).
```
{
  "id": "jmeterd-test",
  "args": [],
  "variables": {},
  "content": {
    "type": "string"
  }
}
```
4. You need to provide the `RUNNER_SCRAPPERENABLED`, `RUNNER_SSL` and `RUNNER_DATADIR` environment variables and run the Executor using the `make run run_args="-f|--file <path>"` make command where `-f|--file <path>` argument is the path to the `execution.json` file you created in step 3.
```
RUNNER_SCRAPPERENABLED=false RUNNER_SSL=false RUNNER_DATADIR="./data" make run run_args="-f temp/execution.json"
```

#### Execution JSON

Execution JSON stores information required for an Executor to run the configured tests.

Breakdown of the Execution JSON:
```
{
   "args": ["-n", "-t", "test.jmx"],
   "variables": {
      "example": {
         "type": "basic", 
         "name": "example", 
         "value": "some-value"
     }
   },
   "content": {
      "type": "string"
   }
}
```
- **args** - An array of strings which will be passed to JMeter as arguments.
  - `example: ["-n", "-t", "test.jmx"]`
- **variables** - The map of variables which will be passed to JMeter as arguments.
  - example: `{"example": {"type": "basic", "name": "example", "value": "some-value"}}`
- **content.type** - Used to specify that JMeter XML is provided as a text file.

#### Environment Variables
```
RUNNER_SSL=false                  # used if storage backend is behind HTTPS, should be set to false for local development
RUNNER_SCRAPPERENABLED=false      # used to enable/disable scrapper, should be set to false for local development
RUNNER_DATADIR=<path-to-data-dir> # path to the data/ directory where JMeter will run and store results
```

## Testing in Kubernetes

### Prerequisites

- Kubernetes cluster with Testkube installed (the best practice is to install it in the testkube namespace).

### Guide

After validating locally that the Executor changes work as expected, the next step is to test whether Testkube can successfully schedule a Test using the new Executor image.

:::note
The following commands assume that Testkube is installed in the `testkube` namespace, if you have it installed in a different namespace, please adjust the `--namespace` flag accordingly.
:::

The following steps need to be executed in order for Testkube to use the new Executor image:

1. Build the new Executor image using the `make docker-build-local` command. By default, the image will be tagged as `kubeshop/testkube-executor-jmeter:999.0.0` unless a `LOCAL_TAG` environment variable is provided before the command.
2. Now you need to make the image accessible in Kubernetes, there are a couple of approaches:
   - *kind* - `kind load docker-image <image-name> --name <kind cluster name>` (e.g. `kind load docker-image testkube-executor-jmeter:999.0.0 --name testkube-k8s-cluster`)
   - *minikube* - `minikube image load <image-name> --profile <minikube profile> (e.g. minikube image load testkube-executor-jmeter:999.0.0 --profile k8s-cluster-test)`
   - *Docker Desktop* - Simply by building the image locally, it becomes accessible in the Docker Desktop Kubernetes cluster.
   - *other* - You can push the image to a registry and then Testkube will pull it into Kubernetes (assuming it has credentials for it, if needed).
3. Edit the Job Template and change the `imagePullPolicy` to `IfNotPresent`.
   - Edit the ConfigMap `testkube-api-server` either by running `kubectl edit configmap testkube-api-server --namespace testkube` or by using a tool like Monokle.
   - Find the `job-template.yml` key and change the `imagePullPolicy` field in the `containers` section to `IfNotPresent`.
4. Edit the Executors configuration and change the base image to use the newly created image:
   - Edit the ConfigMap `testkube-api-server` either by running `kubectl edit configmap testkube-api-server --namespace testkube` or by using a tool like Monokle.
   - Find the `executors.json` key and change the `executor.image` field to use the newly created image for the JMeter Executor (`name` field is `jmeter-executor`).
5. Restart the API Server by running `kubectl rollout restart deployment testkube-api-server --namespace testkube`.

Testkube should now use the new image for the Executor and you can schedule a Test with your preferred method.

