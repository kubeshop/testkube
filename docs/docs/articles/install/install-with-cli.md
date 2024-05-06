# Install Testkube with the CLI

The Testkube CLI includes installation commands to help you set up Testkube for your environment. You can choose from one of our built-in configuration profiles (see below) and the CLI will help you with the last-mile configuration to finalise your setup. You can find instructions on how to install the CLI [here][install-cli].

## Deploy an on-prem demo

Our demo profile bootstraps Testkube’s control plane (dashboard) and agent within the same namespace. It will also create an admin user, organisation and environment.

You will be asked for a license key which you can request for free, no credit card required. You can get the license at https://testkube.io/download

```
testkube init demo
```

Once deployed, use `testkube dashboard` to conveniently access all services on your localhost.


## Deploy an agent that will connect to a control plane

The agent profile installs an agent that will join a control plane running within a different cluster or namespace. The agent acts as a test runner for your organisation’s environment. You can install multiple agents as seen in [the Testkube On-Prem Federated reference architecture][architecture-federated].

You will be asked for an agent token which you can find in your Testkube dashboard.

```
testkube init agent
```


## Deploy the open-source, standalone agent

The standalone-agent profile installs the agent that functions on its own without a control plane (no dashboard available in this mode). It allows you to use the test orchestration engine through the CLI and Custom Resource Definitions.

The standalone-agent is fully open-sourced under a MIT license [on GitHub](https://github.com/kubeshop/testkube).

```
testkube init
```


## Deploy other profiles

You can find all available profiles by running `testkube init --help`. Each profile will interactively ask you the information it needs or you can use `testkube init <profile> --help` to run non-interactively by passing in the required flags.

The following built-in configuration profiles are currently available:

- **demo**: similar to the default profile, but it will configure a default user, organisation and admin to try out Testkube On Prem within minutes.
- **agent**: enables components to run the agent joining a control plane. You can use this profile after creating an environment.
- **standalone-agent**: enables components to run the agent without a control plane. This profile is completely open-source and allows you to run tests with the CLI and CRDs.

<!-- - **default:** enables both the control plane and an agent running within the same namespace. This profile is recommended to get started with light to medium workloads. You can view your test definition and executions within the dashboard.
- **minimal:** enables the control plane without any agent. You will use the profile for advanced setups where agent(s) will run in one or more different clusters or namespaces. Learn more by reading our reference architectures. -->

[install-cli]: /articles/install/cli
[request-license]: https://testkube.io/download
[architecture-federated]: https://deploy-preview-5346--testkube-docs-preview.netlify.app/articles/install/reference-architectures#testkube-on-prem-federated
