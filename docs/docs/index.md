---
sidebar_position: 0
sidebar_label: Welcome
---

# ![Testkube](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

Welcome to Testkube - your somewhat opinionated and friendly Kubernetes testing framework!

<p align="center">
    <img src="https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-intro.gif" />
</p>

Testkube decouples test artifacts and execution from CI/CD tooling. Tests are meant to be part of a cluster's state and can be executed as needed:

- Manually via kubectl CLI.
- Externally triggered via API (CI, external tooling, etc).
- Automatically on deployment of annotated/labeled services/pods/etc (WIP).

The main Testkube components are:

- Kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with API server.
- API Server - Work orchestrator, Runs executors, gathers execution results.
- Custom Resource Descriptors (CRD) Operator - Watches Testkube Custom Resources (CR), handles changes, communicates with API Server.
- Executors - Run tests defined for specific runner, currently available for [Postman](4-test-types/executor-postman.md), [Cypress](4-test-types/executor-cypress.md), [K6](4-test-types/executor-k6.md) and [Curl](4-test-types/executor-curl.md).
- Results DB - For centralized test results management.
- A simple browser-based [User Interface](3-using-testkube/UI.md) for monitoring test results.

Testkube attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD pipelines.
- Make it easy to orchestrate and run any kinds of tests - functional, load/performance, security, compliance, etc.,
  in your clusters, without having to wrap them in docker-images or provide network access.
- Make it possible to decouple test execution from build processes, allowing engineers to run specific tests whenever needed.
- Centralize all test results in a consistent format for actionable QA analytics.
- Provide a modular architecture for adding new types of test scripts and executors.

## **Getting Started**

Check out the [Installation](1-installing.md) and [Getting Started](2-getting-started.md) guides to set up Testkube and 
run your first tests!

<!---<iframe width="560" height="315" src="https://www.youtube.com/embed/rWqlbVvd8Dc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe> --->

## **Blog Posts**

Check out our blog posts that highlight Testkube functionality:

- [Testkube Release v1.5](https://kubeshop.io/blog/testkube-v15-release-notes) - August 29, 2022

## **Questions or Comments?**

What do you think of Testkube? We'd LOVE to hear from you! Please share your experiences and, of course, ideas on how we can make it better. Feel free to reach out on our [Discord server](https://discord.gg/uNuhy6GDyn).
