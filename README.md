```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                                 /kʌb tɛst/ by Kubeshop
```

<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347

Known vulnerabilities: ![kubtest](https://snyk.io/test/github/kubeshop/kubtest/badge.svg)
![kubtest-operator](https://snyk.io/test/github/kubeshop-operator/kubtest/badge.svg)
![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)
-->
                                                           
# Welcome to Kubtest - your friendly Kubernetes testing framework!

Kubtest decouples test artefacts and execution from CI/CD tooling; tests are meant to be part of your
clusters state and can be executed as needed:

- Manually via kubectl cli
- Externally triggered via API (CI, external tooling, etc)
- Automatically on deployment of annotated/labeled services/pods/etc (WIP)

Main Kubtest components are:

- kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with
- API Server - work orchestrator, runs executors, gather execution results
- CRDs Operator - watch Kubtest CR, handles changes communicates with API Server
- Executors - runs tests defined for specific runner
- Results DB - for centralized test results mgmt

Kubtest attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD  pipelines
- Make it easy to run any kind of tests - functional, load/performance, security, compliance, etc. - in your clusters, 
  without having to wrap them in docker-images or providing network access
- Make it possible to decouple test execution from build processes; engineers should be able to run specific tests whenever needed
- Centralize all test results in a consistent format for "actionable QA analytics"
- Provide a modular architecture for adding new types of test scripts and executors

## Getting Started

Check out the [Installation](https://kubeshop.github.io/kubtest/installing/) and
[Getting Started](https://kubeshop.github.io/kubtest/getting-started/) guides to set up Kubtest and
run your first tests!

# Documentation

Is available at [https://kubeshop.github.io/kubtest](https://kubeshop.github.io/kubtest)

# Contribution to project

Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

