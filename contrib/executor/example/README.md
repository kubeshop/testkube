![TestKube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/logo-dark-text-full.png)

# Welcome to TestKube Executor example

TestKube Executor eample is simple test which checks if `GET` request for URI returns `200 OK` status - it's purpose was for showing how to extend testkube with custom executor. 

# What is executor

Executor is nothing more than program wrapped into Docker container which gets json (testube.Execution) OpenAPI based document, and returns stream of json output lines (testkube.ExecutorOutput) - each output line is simply wrapped in this JSON, like in structured logging idea. 


# Issues and enchancements 

Please follow to main TestKube repository for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

## Running executor example

4. Build and push dockerfile to some repository

5. Register Executor Custom Resource in your cluster 

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
  volume_mount_path: /mnt/artifacts-storage
  volume_quantity: 10Gix

```

Set up volumes as in following example if you want to use artifacts storage (can be downloaded later in dashboard or by `kubectl testkube` plugin)


## Other examples

- [Executor template](https://github.com/kubeshop/testkube-executor-template) - was used to create this example
- [Postman executor](https://github.com/kubeshop/testkube-executor-postman)
- [Cypress executor](https://github.com/kubeshop/testkube-executor-cypress)
- [Curl executor](https://github.com/kubeshop/testkube-executor-curl)

