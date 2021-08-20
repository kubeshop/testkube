```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                               /kjuːb tɛst/ by Kubeshop
```

Welcome to Kubtest - your friendly Kubernetes testing framework!

Kubetest decouples test artefacts and execution from CI/CD tooling; tests are meant to be part of your
clusters state and can be executed as needed:

- Manually via cli
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

Check out the [Installation](installing.md) and [Getting Started](getting-started.md) guides to set up Kubtest and 
run your first tests!