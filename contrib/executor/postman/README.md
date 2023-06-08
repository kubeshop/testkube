![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)
                                                           
# Welcome to testkube Postman Executor

Kubetest Postman Executor Job Agent [testkube](https://testkube.io)

# Issues and enchancements 

Please follow to main testkube repository for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

## Details 

Agent is wrapped as Kubernetes Job on new test execution.

Input: JSON - testkube.Execution as input, and JSON structured log as output.

## Passing secrets
It's possible to pass data from kubernetes secrets attached to each executed test.
Current implementation assumes that the environment variable RUNNER_SECRET_ENV{n} contains postman env file data.
It's automatically processed and added to existing test execution env values.
These secrets envs are defined when the test execution is initiated.

## Architecture

Look at [architecture diagrams in docs](https://docs.testkube.io/articles/architecture)
