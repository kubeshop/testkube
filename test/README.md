# Test directory overview

This directory contains 200+ structured TestWorkflows, serving a dual purpose: assuring the quality of Testkube and its features, and providing a set of real-life examples. It consists of several dozen directories for popular testing tools with example projects, and various TestWorkflows covering different approaches, use cases, and Testkube functionalities. In addition to that, it includes synthetic workflows validating edge cases and failure scenarios. There are also Testkube-specific workflows used internally, such as E2E tests, installation tests, Wizard example validation, and more.

## Directory structure

The directory is organized by testing tool or purpose. Each top-level "tool directory" (e.g. `playwright`, `postman`, `k6`, `cypress`, etc.) contains example projects or tests, and corresponding `TestWorkflow` definitions (usually located in the `crd-workflow` subdirectory).

Some directories (like `junit-pregenerated-reports`) include static test data for specific scenarios. Others (like `special-cases`) contain synthetic workflows for validating engine behavior and edge cases.


```
...
├── curl
│   └── crd-workflow # workflows
├── cypress
│   ├── crd-workflow # workflows
|   ... # Cypress projects for different versions
│   ├── cypress-12
│   ├── cypress-13
│   └── cypress-14
...
├── junit-pregenerated-reports # additional custom cases for junit reports (mainly edge-cases)
│   ├── crd-workflow
│   ├── high-level-failure.xml
│   ├── high-level-testcase-both-error-and-failure.xml
│   └── high-level-without-testcases.xml
...
├── legacy # Executors, Tests, TestSuites and other legacy things
...
├── postman
│   ├── crd-workflow # workflows
|   ... # example Postman collections - including variants and negative (expected failure)
│   ├── postman-executor-smoke-negative.postman_collection.json 
│   ├── postman-executor-smoke-without-envs.postman_collection.json
│   ├── postman-executor-smoke.postman_collection.json
├── special-cases # Special cases and Edge-cases
├── testkube # Testkube-specific workflows - installation tests, E2E tests, etc.
...
```

### Special cases and Edge-cases
The `special-cases` directory contains synthetic workflows created to verify specific engine behaviors, failure conditions, and advanced features that are not entirely covered by standard workflows.

`Special-cases` suite include workflows validating ENV variable resolution and overrides, retries, conditional and optional steps, shared volumes, security contexts, expressions, etc. 
`Expected-fail` suite includes scenarios such as incorrect configuration, timeouts, OOMKilled containers, failed readiness probes, invalid template usage, and parallel execution errors. These workflows are used to ensure that the Testkube engine handles edge cases predictably and fails gracefully when expected.

```
├── special-cases
│   ├── edge-cases-expected-fails-additional.yaml # Additional expected failures for more custom cases (including highly parallelized)
│   ├── edge-cases-expected-fails.yaml # Various expected failures - wrong configs, limits (timeouts, OOMKilled), conditions, etc.
│   ├── edge-cases-random.yaml
│   ├── file-read-write.yaml
│   ├── large.yaml # large logs scenarios
│   ├── special-cases-additional.yaml
│   └── special-cases.yaml # Overrides (ENV, workingDir), conditions, retries, shared volumes, securitycontext runAsUser/runAsGroup
```

### Testkube-specific workflows

`testkube` directory contains Testkube-specific workflows like Installation tests, OSS-specific tests, and E2E tests.

```
├── testkube
│   ├── installation-tests
│   ├── oss-tests
│   ├── runner-targets
│   └── ui-e2e
```


### Suites

```
suites
├── artillery-workflow.yaml
├── cron # CRON and other triggers - triggering specific Test Workflow suites
├── curl-workflow.yaml
├── cypress-workflow.yaml
├── full-smoke.yaml
...
├── playwright-workflow.yaml
├── postman-workflow.yaml
...
├── small-smoke.yaml # Small smoke suite - providing basic feedback early
├── special-cases # special cases and expected-fail suites
│   ├── edge-cases-expected-fails.yaml
│   └── special-cases.yaml
├── testkube-installation-tests-workflow.yaml
├── testkube-multiagent-targets-workflow.yaml # Suite for validating multi-agent feature
├── testkube-oss-workflow.yaml # OSS-specific suite
├── tools-preview-images.yaml # Suite with preview/latest images for specific tools - providing early feedback
...
```

### Triggers

```
suites
├── cron # CRON and other triggers - environment-specific, trigger specific suites based on environment
│   ├── dev # various DEV cron triggers + trigger-small-suite-on-deployment.yaml triggering after deployments
│   │   ├── cloud-ui-e2e.yaml
│   │   ├── edge-cases-special-cases-suite-cron.yaml
│   │   ├── full-smoke-cron.yaml
│   │   ├── prod-healthcheck.yaml
│   │   ├── small-smoke-cron.yaml
│   │   ├── testkube-installation-tests-cron.yaml
│   │   ├── testkube-multiagent-targets-workflow-cron.yaml
│   │   ├── testkube-oss-tests-cron.yaml
│   │   ├── tools-preview-images-cron.yaml
│   │   ├── trigger-small-suite-on-deployment.yaml # triggers small suite automatically after DEV deployment
│   │   ├── wizard-examples-suite-cron.yaml
│   │   └── wizard-examples-suite.yaml
│   ├── prod
│   │   ├── full-smoke-cron.yaml
│   │   └── small-smoke-cron.yaml
│   └── sandbox
│       └── small-smoke-cron.yaml
```

## Labels
All of the workflows are labeled to simplify filtering them.

The standard workflows are labeled with `core-tests: workflows`.
Tool-specific workflows also have the `tool` label - for example: `tool: postman`
There are additional labels for `artifacts` (`artifacts: "true"`) for the ones using artifacts, and `junit: "true"` for the ones generating JUnit reports.

Additional labels:
- `core-tests: special-cases` - Special cases
- `core-tests: expected-fail` - Expected-fail scenarios
- `core-tests: installation` - Installation tests
- `core-tests: workflow-suite` - Test Workflow "suites"
- `core-tests: workflow-suite-trigger` - suite triggers