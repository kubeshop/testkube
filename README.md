```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                                 /kʌb tɛst/ by Kubeshop
```

<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347

Known vulnerabilities: [![Kubtest](https://snyk.io/test/github/kubeshop/kubtest/badge.svg)](https://snyk.io/test/github/kubeshop/kubtest)
[![kubtest-operator](https://snyk.io/test/github/kubeshop/kubtest-operator/badge.svg)](https://snyk.io/test/github/kubeshop/kubtest-operator)
[![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)](https://snyk.io/test/github/kubeshop/helm-charts)
-->
                                                           
# Welcome to Kubtest - your friendly Kubernetes testing framework!

Kubtest decouples test artefacts and execution from CI/CD tooling; tests are meant to be part of your
clusters state and can be executed as needed:

- Manually via kubectl cli
- Externally triggered via API (CI, external tooling, etc)
- Automatically on deployment of annotated/labeled services/pods/etc (WIP)

Main Kubtest components are:

- kubectl Kubtest plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with
- API Server - work orchestrator, runs executors, gather execution results
- [CRDs Operator](https://github.com/kubeshop/kubtest-operator) - watches Kubtest CR, handles changes, communicates with API Server
- Executors - runs tests defined for specific runner
  - [Postman Executor](https://github.com/kubeshop/kubtest-executor-postman) - runs Postman Collections
  - [Cypress Executor](https://github.com/kubeshop/kubtest-executor-cypress) - runs Cypress Tests
  - [Curl Executor](https://github.com/kubeshop/kubtest-executor-curl) - runs simple Curl commands
  - [Executor Template](https://github.com/kubeshop/kubtest-executor-template) - for creating your own executors
- Results DB - for centralized test results aggregation and analysis
- [Kubtest Dashboard](https://github.com/kubeshop/kubtest-dashboard) - standalone web application for viewing real-time Kubtest test results

Kubtest attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD  pipelines
- Make it easy to orchestrate and run any kinds of tests - functional, load/performance, security, compliance, etc. - 
  in your clusters, without having to wrap them in docker-images or providing network access
- Make it possible to decouple test execution from build processes; engineers should be able to run specific tests whenever needed
- Centralize all test results in a consistent format for "actionable QA analytics"
- Provide a modular architecture for adding new types of test scripts and executors

## Getting Started

Check out the [Installation](https://kubeshop.github.io/kubtest/installing/) and
[Getting Started](https://kubeshop.github.io/kubtest/getting-started/) guides to set up Kubtest and
run your first tests!

# Discord

Don't hesitate to say hi to the team and ask questions on our [Discord server](https://discord.gg/uNuhy6GDyn).

# Documentation

Is available at [https://kubeshop.github.io/kubtest](https://kubeshop.github.io/kubtest)

## Contribution to project

Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

# Feedback 

Whether it helps you or not - we'd LOVE to hear from you.  Please let us know what you think and of course, how we can make it better.
