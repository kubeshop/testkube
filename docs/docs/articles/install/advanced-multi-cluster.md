# Multicluster

Testkube can federate multiple clusters.
Conceptually, each cluster maps to an environment within Testkube.
You will require a pro plan to deploy multiple Testkube agents.

## Deploy an agent that will join Testkube

You can add another agent to an existing Testkube deployment within a couple of minutes. Get started by going to the dashboard and create a new environment. Afterwards it will show you a command that can be used to bootstrap the agent in another cluster. The command looks as follows:

```
testkube pro init
```

#### Multiple agents within the same cluster

It's possible to install multiple agents within the same cluster. This requires modified values for the second agent to prevent creating cluster-wide objects twice which is disallowed by Kubernetes. Make the following changes to the values of **the second agent**:

```diff
testkube-operator:
-  enabled: true
+  enabled: false
```

## Migrating a standalone agent that will join Testkube

You can also migrate from the open-source Testkube standalone agent to a federated agent.
The following command will walk you through the migration process. Once completed, data will
be send Testkube's unified control plane.

```
testkube pro connect
```

To complete the procedure, you will finally have to [set your CLI Context to talk to Testkube][cli-context].

:::danger
Currently historical logs and artifacts are not uploaded to the control plane.
We plan to add this in the near future. Please [contact us][contact] if this is important to you.
:::

## Deploy a control plane without an agent

By default, Testkube will create an environment within the same namespace as the control plane. You can choose to

Within the Helm values.yaml make the following changes:

```diff
testkube-agent:
-  enabled: true
+  enabled: false

testkube-cloud-api:
  api:
    features:
-      bootstrapEnv: "my-first-environment"
-      bootstrapAgentTokenSecretRef: "testkube-default-agent-token"
```

Once started, you can [deploy agents that will join your control plane as described above][deploy-agent].

[deploy-agent]: /articles/install/advanced-multi-cluster#deploy-an-agent-that-will-join-your-control-plane
[contact]: https://testkube.io/contact
[cli-context]: /testkube-pro/articles/managing-cli-context
