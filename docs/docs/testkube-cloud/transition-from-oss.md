# Transition from Testkube OSS

To migrate Testkube OSS to Cloud you need to install Testkube in Cloud Agent mode. Testkube Cloud Agent is the Testkube engine for managing test runs into your cluster. It sends data to Testkubes Cloud Servers. It's main responsibility is to manage test workloads and to get insight into Testkube resources stored in the cluster.


## Installing the Agent

Please follow the [install steps](installing-agent.md) to get started using the Testkube Agent.

## Setting the Testkube CLI contenxt to the agent

Please follow the [install steps](managing-cli-context.md) to configure your Testkube CLI in Cloud mode.


## Migrating the Testkube Resources

Currently there is no automatic migration tool for existing Testkube OSS resources. But we have plan for it in incoming releases.
