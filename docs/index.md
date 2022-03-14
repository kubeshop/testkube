![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

Welcome to Testkube - your somewhat opinionated and friendly Kubernetes testing framework!

<p align="center">
    <img src="https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-intro.gif">
</p>

Testkube decouples test artifacts and execution from CI/CD tooling. Tests are meant to be part of a cluster's state and can be executed as needed:

- Manually via kubectl CLI.
- Externally triggered via API (CI, external tooling, etc).
- Automatically on deployment of annotated/labeled services/pods/etc (WIP).

The main Testkube components are:

- Kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with API server.
- API Server - Work orchestrator, Runs executors, gathers execution results.
- Custom Resource Descriptors (CRD) Operator - Watches Testkube Custom Resources (CR), handles changes, communicates with API Server.
- Executors - Run tests defined for specific runner, currently available for [Postman](executor-postman.md), [Cypress](executor-cypress.md) and [Curl](executor-curl.md).
- Results DB - For centralized test results management.
- A simple browser-based [Dashboard](dashboard.md) for monitoring test results.

Testkube attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD pipelines.
- Make it easy to orchestrate and run any kinds of tests - functional, load/performance, security, compliance, etc.,
  in your clusters, without having to wrap them in docker-images or provide network access.
- Make it possible to decouple test execution from build processes, allowing engineers to run specific tests whenever needed.
- Centralize all test results in a consistent format for actionable QA analytics.
- Provide a modular architecture for adding new types of test scripts and executors.

## **Getting Started**

Check out the [Installation](installing.md) and [Getting Started](getting-started.md) guides to set up Testkube and 
run your first tests!

<!---<iframe width="560" height="315" src="https://www.youtube.com/embed/rWqlbVvd8Dc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe> --->

## **Questions or Comments?**

What do you think of Testkube? We'd LOVE to hear from you! Please share your experiences and, of course, ideas on how we can make it better. Feel free to reach out on our [Discord server](https://discord.gg/uNuhy6GDyn).
