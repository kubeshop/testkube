<p align="center">  
  <img style="width:66%" src="assets/testkube-color-white.png#gh-dark-mode-only" alt="Testkube Logo Light"/>
  <img style="width:66%" src="assets/testkube-color-dark.png#gh-light-mode-only" alt="Testkube Logo Dark" />
</p>

<p align="center">
  <a href="https://github.com/kubeshop/testkube/releases"><img title="Release" src="https://img.shields.io/github/v/release/kubeshop/testkube"/></a>
  <a href=""><img title="Downloads" src="https://img.shields.io/github/downloads/kubeshop/testkube/total.svg"/></a>
  <a href=""><img title="Go version" src="https://img.shields.io/github/go-mod/go-version/kubeshop/testkube"/></a>
  <a href=""><img title="Docker builds" src="https://img.shields.io/docker/automated/kubeshop/testkube-api-server"/></a>
  <a href=""><img title="Code builds" src="https://img.shields.io/github/workflow/status/kubeshop/testkube/Code%20build%20and%20checks"/></a>
  <a href=""><img title="mit licence" src="https://img.shields.io/badge/License-MIT-yellow.svg"/></a>
  <a href="https://github.com/kubeshop/testkube/releases"><img title="Release date" src="https://img.shields.io/github/release-date/kubeshop/testkube"/></a>
  <a href="https://contribute.design/kubeshop/testkube"><img title="Design contributions welcome" src="https://contribute.design/api/shield/kubeshop/testkube"/></a>
</p>

<p align="center">
  <a href="https://testkube.io">Website</a>&nbsp;|&nbsp;
  <a href="https://docs.testkube.io">Documentation</a>&nbsp;|&nbsp; 
  <a href="https://twitter.com/testkube_io">Twitter</a>&nbsp;|&nbsp; 
  <a href="https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email">Slack</a>&nbsp;|&nbsp; 
  <a href="https://kubeshop.io/category/testkube">Blog</a>
</p>


<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347
Known vulnerabilities: [![Testkube](https://snyk.io/test/github/kubeshop/testkube/badge.svg)](https://snyk.io/test/github/kubeshop/testkube)
[![testkube-operator](https://snyk.io/test/github/kubeshop/testkube-operator/badge.svg)](https://snyk.io/test/github/kubeshop/testkube-operator)
[![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)](https://snyk.io/test/github/kubeshop/helm-charts)
-->

# Welcome to Testkube!

Testkube decouples test orchestration and execution from your CI/CD/GitOps tooling and provides a centralized platform 
for running any kind of tests at scale across your entire application infrastructure. 

Testkube breaks down Test Execution into 5 steps:

1. **Define** - Use Test Workflows to configure executions of your current testing tools or scripts. 
  Orchestrate multiple Workflows to build complex Suites for System Testing - [Read More](https://docs.testkube.io/articles/defining-tests).
2. **Trigger** - Trigger tests through the API/CLI, from your existing CI/CD/GitOps workflows, using fixed schedules or 
  by listening to Kubernetes Events or creating execution CRDs - [Read More](https://docs.testkube.io/articles/triggering-tests).
3. **Scale** - Leverage Kubernetes native scalability functionality to scale your test executions 
  across distributed nodes for both load and functional testing with popular tools like K6, Playwright, JMeter and Cypress - [Read More](https://docs.testkube.io/articles/running-scaling-tests).
4. **Troubleshoot** - Testkube can collect any logs and artifacts (videos, reports, etc.) produced by your testing tools 
  and scripts during test execution and make these available through the CLI or UI - [Read More](https://docs.testkube.io/articles/troubleshooting-tests).
5. **Report** - Testkube Test Insights allow you to create both operational and functional reports for all your test executions 
  to help you improve testing efforts and activities over time - [Read More](https://docs.testkube.io/articles/analyzing-results).

### Getting Started

There are several ways to get started with Testkube:

- The [Quickstart](https://docs.testkube.io/articles/tutorial/quickstart) is the easiest way to set up 
  Testkube and run your first tests
- The [Helm Chart Installation](https://docs.testkube.io/articles/install/install-with-helm) gives you more 
  control over the installed components.

Check out the [Deployment Architectures](https://docs.testkube.io/articles/install/deployment-architectures) document to learn
more about different ways to deploy and run Testkube.

### Documentation

Extensive documentation is available at [docs.testkube.io](https://docs.testkube.io).

## Contributing

Shout-out to our contributors 🎉 - you're great!

- ⭐️ [@lreimer](https://github.com/lreimer) - [K6 executor](https://github.com/kubeshop/testkube-executor-k6) [Gradle executor](https://github.com/kubeshop/testkube-executor-gradle) [Maven executor](https://github.com/kubeshop/testkube-executor-maven)
- ⭐️ [@jdborneman-terminus](https://github.com/jdborneman-terminus) - [Ginkgo executor](https://github.com/kubeshop/testkube-executor-ginkgo) 
- ️⭐️ [@abhishek9686](https://github.com/abhishek9686)
- ⭐️ [@ancosma](https://github.com/ancosma)
- ⭐️ [@Stupremee](https://github.com/Stupremee)
- ⭐️ [@artem-zherdiev-ingio](https://github.com/artem-zherdiev-ingio)
- ⭐️ [@chooco13](https://github.com/chooco13) - [Playwright executor](https://github.com/kubeshop/testkube-executor-playwright)


Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

# Feedback

Whether it helps you or not - we'd LOVE to hear from you.  Please let us know what you think and of course, how we can make it better
Please join our growing community on [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email).
