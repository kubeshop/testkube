import CICDComparison from "../img/cicd-comparison.png";

# Benefits of using Testkube

<img src={CICDComparison} />

Whether you want to simplify your company's DevOps workflows or empower your QA and Testing teams, Testkube provides your teams the power to:

## Run your tests inside your cluster

Testkube runs your tests inside your Kubernetes cluster, and not from a CI pipeline. This is a huge networking security benefit because you don't need to expose your cluster to the world to be able to test its application. 

## GitOps Friendly Testing Strategy

Testkube is Kubernetes-native, meaning all your tests are defined using Kubernetes Custom Resources. This allows your tests to be part of your Infrastructure as Code. With Testkube you can use GitOps tools like ArgoCD and Flux to create and manage your tests.

## Make your Tests Kubernetes Native

Your tests are native to Kubernetes as Testkube uses Custom Resources to manage the definitions and execution of your tests.

## Centralized reporting and storage of test artifacts

Testkube can run any test tool that you're using, but the best part is that you'll stop having test results that are spread around multiples systems and you can have a single place inside **your** cluster where all test results are saved and reported with a common format.

## Run Tests on demand

Currently, if you want to re-run your test, you'll probably be re-triggering your entire CI/CD pipeline, and you'll probably have already lost 10 minutes of your life doing it. 

Testkube allows you to run and re-run your tests on command or automatically using: 

- ✨Automatically on deployment of annotated/labeled Kubernetes objects (services, pods, etc)
- ⏲️ On a schedule
- 🧑‍💻 Manually via Testkube's CLI or Open Source Dashboard
- ⚡ Externally triggered via API
