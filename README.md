```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                                 /kʌb tɛst/ by Kubeshop
```

<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347

Known vulnerabilities: [![TestKube](https://snyk.io/test/github/kubeshop/testkube/badge.svg)](https://snyk.io/test/github/kubeshop/testkube)
[![testkube-operator](https://snyk.io/test/github/kubeshop/testkube-operator/badge.svg)](https://snyk.io/test/github/kubeshop/testkube-operator)
[![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)](https://snyk.io/test/github/kubeshop/helm-charts)
-->
                                                           
# Welcome to TestKube - your friendly Kubernetes testing framework!

TestKube decouples test artefacts and execution from CI/CD tooling; tests are meant to be part of your
clusters state and can be executed as needed:

- Manually via kubectl cli
- Externally triggered via API (CI, external tooling, etc)
- Automatically on deployment of annotated/labeled services/pods/etc (WIP)

Main TestKube components are:

- kubectl TestKube plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with
- API Server - work orchestrator, runs executors, gather execution results
- [CRDs Operator](https://github.com/kubeshop/testkube-operator) - watches TestKube CR, handles changes, communicates with API Server
- Executors - runs tests defined for specific runner
  - [Postman Executor](https://github.com/kubeshop/testkube-executor-postman) - runs Postman Collections
  - [Cypress Executor](https://github.com/kubeshop/testkube-executor-cypress) - runs Cypress Tests
  - [Curl Executor](https://github.com/kubeshop/testkube-executor-curl) - runs simple Curl commands
  - [Executor Template](https://github.com/kubeshop/testkube-executor-template) - for creating your own executors
- Results DB - for centralized test results aggregation and analysis
- [TestKube Dashboard](https://github.com/kubeshop/testkube-dashboard) - standalone web application for viewing real-time TestKube test results

TestKube attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD  pipelines
- Make it easy to orchestrate and run any kind of tests - functional, load/performance, security, compliance, etc. - 
  in your clusters, without having to wrap them in docker-images or providing network access
- Make it possible to decouple test execution from build processes; engineers should be able to run specific tests whenever needed
- Centralize all test results in a consistent format for "actionable QA analytics"
- Provide a modular architecture for adding new types of test scripts and executors

## Getting Started

Check out the [Installation](https://kubeshop.github.io/testkube/installing/) and
[Getting Started](https://kubeshop.github.io/testkube/getting-started/) guides to set up TestKube and
run your first tests!

# Discord

Don't hesitate to say hi to the team and ask questions on our [Discord server](https://discord.gg/uNuhy6GDyn).

# Documentation

Is available at [https://kubeshop.github.io/testkube](https://kubeshop.github.io/testkube)

## Contributing

Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

# Feedback 

Whether it helps you or not - we'd LOVE to hear from you.  Please let us know what you think and of course, how we can make it better.
