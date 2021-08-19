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

- A kubectl plugin for creating/running tests
- Custom resource definitions and corresponding controllers/operators for defining test scripts - WIP
- Extension mechanism that allows 3rd party tool providers to add support for their test scripts
- Integration with 3rd party tools for result reporting/analysis (prometheus, etc.) - WIP
- Custom controller/operator that can be configured to run specific tests based on events/annotations/etc - WIP

Kubtest attempts to:

- Avoid vendor lock-in for CI/CD test orchestration and execution pipelines
- Make it easy to run any kind of tests - functional, load/performance, security, compliance, etc. - in your clusters, without having to wrap them in docker-images or providing network access
- Provide a modular architecture for adding new types of test scripts and executors

Check out the [Installation](installing.md) and [Gettin Started](getting-started.md) guides to setup Kubtest and 
run your first tests!