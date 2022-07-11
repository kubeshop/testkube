![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to TestKube Template Executor

TestKube Template Executor is a test executor skeleton for [TestKube](https://testkube.io).  
You can use it as basic building blocks for creating a new executor.

# What is an Executor?

Executor is nothing more than a program wrapped into Docker container which gets JSON (testube.Execution) OpenAPI based document as an input and returns a stream of JSON output lines (testkube.ExecutorOutput), where each output line is simply wrapped in this JSON, similar to the structured logging idea. 


# Issues and enchancements 

Please follow the main [TestKube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

## Implemention in several steps:

1. Create new repo on top of this template 
2. Change `go.mod` file with your path (just replace `github.com/kubeshop/testkube-executor-template` project-wise with your package path) 
3. Implement your own Runner on top of [runner interface](https://github.com/kubeshop/testkube/blob/main/pkg/runner/interface.go
4. Change Dockerfile - use base image of whatever test framework/library you want to use
5. Build and push dockerfile to some repository
6. Register Executor Custom Resource in your cluster 

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: postman-executor
  namespace: testkube
spec:
  executor_type: job
  image: kubeshop/testkube-template-executor:0.0.1
  types:
  - example/test
```


## Architecture

This Executor template offers you basic building blocks to write a new executor based on TestKube 
libraries written in Go programming language, but you're not limited only to Go, you can 
write in any other programming language like Rust, Javascript, Java or Clojure.

The only thing you'll need to do is to follow the OpenAPI spec for input `testkube.Execution` 
(passed as first argument in JSON form) and all output should be JSON lines 
with `testkube.ExecutorOutput` spec.  
You should also have a final `ExecutorOutput` with `ExecutionResult` attached somewhere after successful (or failed) test execution.

Resources: 
- [OpenAPI spec details](https://kubeshop.github.io/testkube/openapi/)
- [Spec in YAML file](https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml)

Go based resources for input and output objects:
- input: [`testkube.Execution`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_execution.go)
- output line: [`testkube.ExecutorOutput`](https://github.com/kubeshop/testkube/blob/main/pkg/api/v1/testkube/model_executor_output.go)


## Examples

- This template repo, which is the simplest one
- [Postman executor](https://github.com/kubeshop/testkube-executor-postman)
- [Cypress executor](https://github.com/kubeshop/testkube-executor-cypress)
- [Curl executor](https://github.com/kubeshop/testkube-executor-curl)


# Testkube 

For more info go to [main testkube repo](https://github.com/kubeshop/testkube)

![Release](https://img.shields.io/github/v/release/kubeshop/testkube) [![Releases](https://img.shields.io/github/downloads/kubeshop/testkube/total.svg)](https://github.com/kubeshop/testkube/tags?label=Downloads) ![Go version](https://img.shields.io/github/go-mod/go-version/kubeshop/testkube)

![Docker builds](https://img.shields.io/docker/automated/kubeshop/testkube-api-server) ![Code build](https://img.shields.io/github/workflow/status/kubeshop/testkube/Code%20build%20and%20checks) ![Release date](https://img.shields.io/github/release-date/kubeshop/testkube)

![Twitter](https://img.shields.io/twitter/follow/thekubeshop?style=social) ![Discord](https://img.shields.io/discord/884464549347074049)
 #### [Documentation](https://kubeshop.github.io/testkube) | [Discord](https://discord.gg/hfq44wtR6Q) 