# kubtest - your friendly Kubernetes testing framework

Kubernetes-native framework for test definition and execution. 

Instead of orchestrating and executing tests with a CI tool (jenkins, travis, circle-ci, GitHub/GitLab, etc),
tests are defined/orchestrated/executed using k8s native concepts (manifests, etc.) and executed either manually via kubectl 
or automatically on external (i.e. CI/CD) or internal triggers (for example when resources are updated in a cluster). 
Results are written to existing tooling (prometheus, etc). 

This decouples test-definition and execution from CI-tooling/pipelines and ensures that tests are run when 
corresponding resources are updated (which could still be part of a CI/CD workflow). 
