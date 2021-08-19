```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                               /kjuːb tɛst/ by Kubeshop
```

![kubtest known vulnerabilities](https://snyk.io/test/github/kubeshop/kubtest/badge.svg)
![kubtest-operator known vulnerabilities](https://snyk.io/test/github/kubeshop-operator/kubtest/badge.svg)
![helm-charts known vulnerabilities](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)
                                                           
# kubtest - your testing friend for your cloud apps


Kubernetes-native framework for definition and execution of tests in a cluster; 

Instead of orchestrating and executing test with a CI tool (jenkins, travis, circle-ci, GitHub/GitLab, etc) tests are defined/orchestrated in the cluster using k8s native concepts (manifests, etc) and executed automatically when target resources are updated in the cluster. Results are written to existing tooling (prometheus, etc). This decouples test-definition and execution from CI-tooling/pipelines and ensures that tests are run when corresponding resources are updated (which could still be part of a CI/CD workflow). 

kubtest components:
- kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with  
- API Server - work orchestrator, runs executors, gather execution results
- CRDs Operator - watch kubtest CR, handles changes communicates with API Server
- Executors - runs tests defined by specific runner, for PoC phase we'll run Postman collection defined in CR.

# Contribution to project

Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

# Installing 

For install instructions [go to install manual](docs/installing.md) 

# Getting started and Usage

For getting started or usage follow 
[getting started docs](docs/getting-started.md) 

# Architecture

Go to [architecture diagrams](docs/architecture.md) for architecture docs

