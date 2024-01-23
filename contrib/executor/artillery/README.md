![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to TestKube Artillery Executor

TestKube Artillery Executor is a test executor for [TestKube](https://testkube.io).  
You can use it to perform load and performance testing for your application running on the cluster.

# What is an Executor?

Executor is nothing more than a program wrapped into Docker container which gets JSON (testube.Execution) OpenAPI based document as an input and returns a stream of JSON output lines (testkube.ExecutorOutput), where each output line is simply wrapped in this JSON, similar to the structured logging idea. 

# Why Artillery ?

Artillery is a modern, powerful & easy-to-use performance testing toolkit. Use it to ship scalable applications that stay performant & resilient under high load.

Artillery prioritizes developer productivity and happiness, and follows the "batteries-included" philosophy.


# Usage

## You need to register and deploy the executor in your cluster

  ```yaml
  apiVersion: executor.testkube.io/v1
  kind: Executor
  metadata:
    name: artillery-executor
    namespace: testkube
  spec:
    image: kubeshop/testkube-executor-artillery:latest
    types:
    - artillery/test
  ```

  ```
  kubectl apply -f examples/artillery-executor.yaml
  ```
## To Create And Run Artillery based tests run following commands
```
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-artillery.git --git-branch main --git-path examples/test.yaml --name artillery-example-test --test-content-type git-file --type artillery/test

```
```
kubectl testkube run test artillery-example-test -f
```
## To Download Artillery Test Report

```
 kubectl-testkube download artifacts  [ Test ExecutionID ] --download-dir [ Destination Directory ]
```


# Issues and enchancements 

Please follow the main [TestKube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)


# Testkube 

For more info go to [main testkube repo](https://github.com/kubeshop/testkube)

![Release](https://img.shields.io/github/v/release/kubeshop/testkube) [![Releases](https://img.shields.io/github/downloads/kubeshop/testkube/total.svg)](https://github.com/kubeshop/testkube/tags?label=Downloads) ![Go version](https://img.shields.io/github/go-mod/go-version/kubeshop/testkube)

![Docker builds](https://img.shields.io/docker/automated/kubeshop/testkube-api-server) ![Code build](https://img.shields.io/github/workflow/status/kubeshop/testkube/Code%20build%20and%20checks) ![Release date](https://img.shields.io/github/release-date/kubeshop/testkube)

![Twitter](https://img.shields.io/twitter/follow/thekubeshop?style=social) ![Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email)
 #### [Documentation](https://docs.testkube.io) | [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email) 