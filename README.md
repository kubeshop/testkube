



[Website](https://testkube.io) |  [Documentation](https://docs.testkube.io) |  [Changelog](https://docs.testkube.io/changelog) |  [Blog](https://testkube.io/blog) |  [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email) |  [LinkedIn](https://www.linkedin.com/company/testkube) |  [X](https://twitter.com/testkubeio)

# Welcome to Testkube!

Testkube is a Test Orchestration Platform for Cloud-Native Applications. It provides a single platform for defining, running and analyzing test executions, using your existing testing tools/scripts, running in your existing infrastructure.

Testkube consists of a **Control Plane** and any number of **Testkube Agents**. The Control Plane exposes a 
Dashboard for easy and centralized access to most Testkube features.

The Testkube Agent (this repo) is **100% Open-Source** and can be deployed standalone without a Control Plane - [Read More](https://docs.testkube.io/articles/open-source).

### Why use Testkube?

- **Run any Tests** : Execute _any_ tests/tools/scripts at scale; API, E2E, Performance, Security, Infrastructure, etc. - [Examples & Guides](https://docs.testkube.io/articles/examples/overview).
- **Trigger Tests whenever needed**: Trigger tests manually, on schedules, from CI/CD/GitOps pipelines, on Kubernetes Events, via the REST API, through MCP, etc. - [Read More](https://docs.testkube.io/articles/triggering-overview).
- **Results and Analytics**: All test results, artifacts, logs and resource-metrics are aggregated for centralized troubleshooting and reporting - [Read More](https://docs.testkube.io/articles/results-overview).
- **Works with your tools**: Integrate with existing tools and infrastructure using [Webhooks](https://docs.testkube.io/articles/webhooks), the [Testkube REST API](https://docs.testkube.io/openapi/overview) or the [MCP Server](https://docs.testkube.io/articles/mcp-overview) - see [Integration Examples](https://docs.testkube.io/articles/integrations).
- **AI Agents**: Build AI Agents for troubleshooting, analysis, remediation, etc - [Read More](https://docs.testkube.io/articles/ai-agents)
- **Enterprise Ready**: SSO/SCIM, RBAC, Teams, Resource-Groups, Audit-logs, etc. - [Read More](https://docs.testkube.io/articles/administration-overview).

### Getting Started with Testkube Open Source

To get started with the open source agent:

- The [Helm or CLI Installation](https://docs.testkube.io/articles/install/standalone-agent#installing-the-standalone-agent) will make it easy to deploy the agent into your target cluster.
- The [Quickstart](https://docs.testkube.io/articles/getting-started-with-open-source) is the easiest way to set up 
Testkube and run your first tests.

Check out the [Testkube Open Source Overview](https://docs.testkube.io/articles/open-source) to learn
more about the open source deployment architecture.

### Getting Started with the Commercial Control Plane

Looking for more than single environment test execution? Do you need orchestration accross clusters, support for different trigger points, and high level reporting and artifact collection? Enterprise may be for your team - there are several ways to get started:

- The [Quickstart](https://docs.testkube.io/articles/tutorial/quickstart/overview) is the easiest way to set up 
Testkube and run your first tests
- The [Helm Chart Installation](https://docs.testkube.io/articles/install/install-with-helm) gives you more control over the installed components.
- The [Feature Comparison](https://docs.testkube.io/articles/install/feature-comparison) page details the differences between Enterprise and Open Source.

Check out the [Installation Overview](https://docs.testkube.io/articles/install/overview) to learn
more about different ways to deploy and run the Testkube Control Plane.

### Documentation

Extensive documentation is available at [docs.testkube.io](https://docs.testkube.io).

### Contributing

Check out our [Contributors Guide](CONTRIBUTING.md) and the [Agent Architecture](ARCHITECTURE.md) to find your way around our codebase and process.

If you want to contribute code, this reading order works well:

1. [CONTRIBUTING.md](CONTRIBUTING.md) - contribution workflow, coding standards, and PR process
2. [DEVELOPMENT.md](DEVELOPMENT.md) - local setup with Tilt and day-to-day development loop
3. [ARCHITECTURE.md](ARCHITECTURE.md) - high-level system design and key code paths

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