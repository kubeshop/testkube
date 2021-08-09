# KubeTest

Kubernetes-native framework for definition and execution of tests in a cluster; 

Instead of orchestrating and executing test with a CI tool (jenkins, travis, circle-ci, GitHub/GitLab, etc) tests are defined/orchestrated in the cluster using k8s native concepts (manifests, etc) and executed automatically when target resources are updated in the cluster. Results are written to existing tooling (prometheus, etc). This decouples test-definition and execution from CI-tooling/pipelines and ensures that tests are run when corresponding resources are updated (which could still be part of a CI/CD workflow). 

# Architecture (C4 Diagrams)
## Info

Diagrams was made with use of C4 diagram technique 
(https://c4model.com/) 



## Diagrams

### System Context 

![Kubetest system context diagram](docs/system_context.png)

### Containers

![Kubetest container diagram](docs/containers.png)

### Components

#### API 

![API](docs/components_api.png)

#### Operator

TODO  add diagram for operator

#### Executor

![Postman-Executor](docs/components_executor.png)


## Building diagrams

To build diagrams you'll need to install plantuml:

```sh
brew install plantuml # on mac
sudo apt-get install -y plantuml # on ubuntu linux 
pacman -S plantuml # on arch linux
```
Next run 
```sh 
make diagrams
```
to generate png files from puml files


TIP: If using vscode there is nice extension for live preview of plantuml files.  


# Minimal components for PoC

- kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with  
- API Server - work orchestrator, runs executors, gather execution results
- CRDs Operator - watch KubeTest CR, handles changes communicates with API Server
- Executors - runs tests defined by specific runner, for PoC phase we'll run 
  Postman collection defined in CR.

## Key components from confluence

- custom crds and corresponding controllers/operators for defining testsuites consisting or supported artefacts
- extension mechanism that allows 3rd party tool providers to add support for their test artefact
- kubectl extension for examining/running tests
- integration with 3rd party tools for result reporting/analysis (prometheus, etc.)
- custom controller/operator that can be configured to run specific tests based on events/annotations/etc



