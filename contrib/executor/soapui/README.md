![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to Testkube SoapUI Executor

Testkube SoapUI Executor is a test executor module for [Testkube](https://testkube.io).

# Details

## Importing the SoapUI Executor into your Testkube cluster

In order to be able to run tests with this executor, it first needs to be imported as a Custom Resource into your Kubernetes cluster.
This can be achieved by cloning the repository and running:

```bash
$ kubectl testkube create executor --image kubeshop/testkube-executor-soapui:latest --types "soapui/xml" --name soapui-executor

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Executor created soapui-executor 🥇
```

## Running a SoapUI test

In order to run a SoapUI test using Testkube, it is necessary to create a Testkube Test.

### Using files as input

Testkube and the SoapUI executor accepts a project file as input.

```bash
$ kubectl testkube create test --file REST-Project-1-soapui-project.xml --type soapui/xml --name example-test

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Test created  / example-test 🥇

```

### Using strings as input

```bash
$ cat REST-Project-1-soapui-project.xml | kubectl testkube create test --type soapui/xml --name example-test-string

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Test created  / example-test-string 🥇

```

## Running the tests

To run the created test, use:

```bash
$ kubectl testkube run test example-test

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : soapui/xml
Name          : example-test
Execution ID  : 624eedd443ed8485ae9289e2
Execution name: illegally-credible-mouse



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 624eedd443ed8485ae9289e2


Use following command to get test execution details:
$ kubectl testkube get execution 624eedd443ed8485ae9289e2

```

### Using params and args in your tests

SoapUI lets you configure your test runs using different parameters. To see all available command line arguments, check the [official SoapUI docs](https://www.soapui.org/docs/test-automation/running-functional-tests/).

When working with Testkube, the way to use the parameters is by using the `kubectl testkube start` command with the `--args` parameter.
An example would be:

```bash
$ kubectl testkube start test -f example-test --args '-I -c "TestCase 1"'

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : soapui/xml
Name          : successful-test
Execution ID  : 625404e5a4cc6d2861193c60
Execution name: currently-amused-pug


Getting pod logs
Execution completed ================================
=
= SOAPUI_HOME = /usr/local/SmartBear/SoapUI-5.7.0
=
================================
SoapUI 5.7.0 TestCase Runner
10:37:37,713 INFO  [DefaultSoapUICore] Creating new settings at [/root/soapui-settings.xml]
10:37:43,567 INFO  [PluginManager] 0 plugins loaded in 36 ms
10:37:43,570 INFO  [DefaultSoapUICore] All plugins loaded
10:37:50,774 INFO  [WsdlProject] Loaded project from [file:/tmp/test-content359342991]
10:37:50,834 INFO  [SoapUITestCaseRunner] Running SoapUI tests in project [REST Project 2]
10:37:50,838 INFO  [SoapUITestCaseRunner] Running TestCase [TestCase 1]
10:37:50,876 INFO  [SoapUITestCaseRunner] Running SoapUI testcase [TestCase 1]
10:37:50,901 INFO  [SoapUITestCaseRunner] running step [1 - Request 1]
10:37:54,180 INFO  [SoapUITestCaseRunner] Assertion [Valid HTTP Status Codes] has status VALID
10:37:54,193 INFO  [SoapUITestCaseRunner] Assertion [Contains] has status VALID
10:37:54,257 INFO  [SoapUITestCaseRunner] Finished running SoapUI testcase [TestCase 1], time taken: 990ms, status: FINISHED
10:37:54,315 INFO  [SoapUITestCaseRunner] TestCase [TestCase 1] finished with status [FINISHED] in 990ms


.
Use following command to get test execution details:
$ kubectl testkube get execution 625404e5a4cc6d2861193c60
```

Usage of the `-I` argument is highly suggested to get cleaner results.

## Reports, plugins and extensions

In order to be able to use reports, add plugins and extensions the way [SoapUI docs](https://www.soapui.org/docs/test-automation/running-in-docker/) describe it, is currently not supported by Testkube.
In case you need this feature, please create an issue in the Testkube repository as described below.

# API

Testkube Executor SoapUI implements [testkube OpenAPI for executors](https://docs.testkube.io/openapi#tag/executor) (look at executor tag)

# Issues and enchancements

Please follow the main [Testkube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)
