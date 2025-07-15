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
  <a href="https://docs.testkube.io/changelog">Changelog</a>&nbsp;|&nbsp;
  <a href="https://testkube.io/blog">Blog</a>&nbsp;|&nbsp;
  <a href="https://hub.docker.com/u/testkube">DockerHub</a>&nbsp;|&nbsp;
  <a href="https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email">Slack</a>&nbsp;|&nbsp; 
  <a href="https://www.linkedin.com/company/testkube">LinkedIn</a>&nbsp;|&nbsp;
  <a href="https://twitter.com/testkube_io">X</a> 
</p>


<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347
Known vulnerabilities: [![Testkube](https://snyk.io/test/github/kubeshop/testkube/badge.svg)](https://snyk.io/test/github/kubeshop/testkube)
[![testkube-operator](https://snyk.io/test/github/kubeshop/testkube-operator/badge.svg)](https://snyk.io/test/github/kubeshop/testkube-operator)
[![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)](https://snyk.io/test/github/kubeshop/helm-charts)
-->

# Welcome to Testkube!

Testkube is a Test Orchestration and Execution Framework for Cloud-Native Applications. 
It provides a single platform for defining, running and analyzing test executions, using 
your existing testing tools/scripts, leveraging your existing CI/CD/GitOps pipelines and 
Kubernetes infrastructure.

Testkube consists of a **Control Plane** and any number of **Testkube Agents**. The Control Plane exposes a 
Dashboard for easy and centralized access to most Testkube features.

The Testkube Agent (this repo) is **100% Open-Source** and can be deployed standalone without a Control Plane - [Read More](https://docs.testkube.io/articles/open-source).

### Why use Testkube?

- **Run any Tests**: Execute any tests/tools/scripts at scale - [Examples & Guides](https://docs.testkube.io/articles/examples/overview).
- **Run Tests whenever needed**: Run tests manually, on schedules, from CI/CD/GitOps pipelines, on Kubernetes Events, etc. - [Read More](https://docs.testkube.io/articles/triggering-overview).
- **Results and Analytics**: Aggregate all test results, artifacts, logs and resource-metrics for centralized troubleshooting and reporting - [Read More](https://docs.testkube.io/articles/results-overview).
- **Works with your tools**: Integrate with existing tools and infrastructure using [Webhooks](https://docs.testkube.io/articles/webhooks) and the [Testkube REST API](https://docs.testkube.io/openapi/overview) - see [Integration Examples](https://docs.testkube.io/articles/integrations).
- **Enterprise Ready**: SSO/SCIM, RBAC, Teams, Resource-Groups, Audit-logs, etc. - [Read More](https://docs.testkube.io/articles/administration-overview).

### Getting Started

There are several ways to get started with Testkube:

- The [Quickstart](https://docs.testkube.io/articles/tutorial/quickstart) is the easiest way to set up 
  Testkube and run your first tests
- The [Helm Chart Installation](https://docs.testkube.io/articles/install/install-with-helm) gives you more control over the installed components.

Check out the [Installation Overview](https://docs.testkube.io/articles/install/overview) to learn
more about different ways to deploy and run Testkube.

### Documentation

Extensive documentation is available at [docs.testkube.io](https://docs.testkube.io).

### Contributing

Shout-out to our contributors 🎉 - you're great!

- ⭐️ [@lreimer](https://github.com/lreimer) - [K6 executor](https://github.com/kubeshop/testkube-executor-k6) [Gradle executor](https://github.com/kubeshop/testkube-executor-gradle) [Maven executor](https://github.com/kubeshop/testkube-executor-maven)
- ⭐️ [@jdborneman-terminus](https://github.com/jdborneman-terminus) - [Ginkgo executor](https://github.com/kubeshop/testkube-executor-ginkgo) 
- ️⭐️ [@abhishek9686](https://github.com/abhishek9686)
- ⭐️ [@ancosma](https://github.com/ancosma)
- ⭐️ [@Stupremee](https://github.com/Stupremee)
- ⭐️ [@artem-zherdiev-ingio](https://github.com/artem-zherdiev-ingio)
- ⭐️ [@chooco13](https://github.com/chooco13) - [Playwright executor](https://github.com/kubeshop/testkube-executor-playwright)

Go to [contribution document](CONTRIBUTING.md) to read more how can you help us 🔥

### Feedback

Whether Testkube helps you or not, we would love to help and hear from you. Please [join us on Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email) to ask questions 
and let us know how we can make Testkube even better!
