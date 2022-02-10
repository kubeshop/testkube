![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/logo-dark-text-full.png)

Welcome to Testkube - your somewhat opinionated and friendly Kubernetes testing framework!

Testkube decouples test artifacts and execution from CI/CD tooling; tests are meant to be part of your clusters state and can be executed as needed:

- Manually via kubectl cli
- Externally triggered via API (CI, external tooling, etc)
- Automatically on deployment of annotated/labeled services/pods/etc (WIP)

Main Testkube components are:

- kubectl plugin - simple - installed w/o 3rd party repositories (like Krew etc), communicates with
- API Server - work orchestrator, runs executors, gather execution results
- CRDs Operator - watch Testkube CR, handles changes communicates with API Server
- Executors - run tests defined for specific runner, currently available for [Postman](executor-postman.md), [Cypress](executor-cypress.md) and [Curl](executor-curl.md)
- Results DB - for centralized test results mgmt
- A simple browser-based [Dashboard](dashboard.md) for monitoring test results

Testkube attempts to:

- Avoid vendor lock-in for test orchestration and execution in CI/CD  pipelines
- Make it easy to orchestrate and run any kinds of tests - functional, load/performance, security, compliance, etc. -
  in your clusters, without having to wrap them in docker-images or providing network access
- Make it possible to decouple test execution from build processes; engineers should be able to run specific tests whenever needed
- Centralize all test results in a consistent format for "actionable QA analytics"
- Provide a modular architecture for adding new types of tests and executors

Check out our Intro video:

<iframe width="560" height="315" src="https://www.youtube.com/embed/rWqlbVvd8Dc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

Check out the [Installation](installing.md) and [Getting Started](getting-started.md) guides to set up Testkube and 
run your first tests!

Whether it helps you or not - we'd LOVE to hear from you.  Please let us know what you think and of course, how we can make it better.
