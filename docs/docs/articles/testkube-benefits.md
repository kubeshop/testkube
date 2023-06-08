import CICDComparison from "../img/cicd-comparison.png";

# Benefits of Using Testkube

<img src={CICDComparison} />

Whether you want to simplify your company's DevOps workflows or empower your QA and Testing teams, Testkube provides your teams the power to:

## Run Your Tests Inside Your Cluster

Testkube runs your tests inside your Kubernetes cluster and not from a CI pipeline. This is a huge networking security benefit because you don't need to expose your cluster to the world to be able to test its application.

## Execute your tests from any CI/CD tool

We decouple test orchestration from your CI/CD pipelines by triggering Testkube’s testing orchestration and execution engine right from within your CI/CD workflow regardless of the tools you use, giving you vendor neutrality and a plethora of options amongst GitLab, GitHub Actions, CircleCI, or a GitOps approach.

## GitOps Friendly Testing Strategy

Testkube is Kubernetes-native, meaning all your tests are defined using Kubernetes Custom Resources. This allows your tests to be part of your Infrastructure as Code. With Testkube you can use GitOps tools like ArgoCD and Flux to create and manage your tests.

## Make Your Tests Kubernetes Native

Your tests are native to Kubernetes as Testkube uses Custom Resources to manage the definitions and execution of your tests.

## Centralized Reporting and Storage of Test Artifacts

Testkube can run any test tool that you're using. The primary advantages of this feature are:

- Test results will not be spread around multiple systems.
- You can have a single place inside **your** cluster where all test results are saved and reported with a common format.

## Run Tests on Demand

Currently, if you want to re-run a test, you'll probably be re-triggering your entire CI/CD pipeline, and you'll probably spend 10 minutes of your life doing it. 

Testkube allows you to run and re-run your tests on command or automatically: 

- ✨Automatically on deployment of annotated/labeled Kubernetes objects (services, pods, etc).
- ⏲️ On a schedule.
- 🧑‍💻 Manually via Testkube's CLI or Open Source Dashboard.
- ⚡ Externally triggered via an API.
