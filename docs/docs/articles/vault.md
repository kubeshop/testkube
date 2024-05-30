# Using Vault

Testkube has not been verified to work with the various ways Vault can be
integrated into a Kubernetes cluster, but we are ready to support enterprise
customers with the specifics of their environment.

For integrations utilizing the [sidecar
injector](https://developer.hashicorp.com/vault/docs/platform/k8s/injector) you
can try to set the appropriate annotations by adapting the example
configurations below for your needs. If you encounter issues please reach out to
our enterprise support.

## Configurations for sidecar injector

With workflows, you can configure pod annotations both per workflow and
globally:

Chart `testkube`

```yaml
global:
    testWorkflows:
        globalTemplate:
            enabled: true
            spec:
                pod:
                    annotations:
                        vault.hashicorp.com/agent-inject-secret-foo: database/roles/app
                        vault.hashicorp.com/agent-inject-secret-bar: consul/creds/app
                        vault.hashicorp.com/role: "app"
```

For the other executors, you can set these annotations globally with:

Chart `testkube`

```yaml
testkube-api:
    jobPodAnnotations:
        vault.hashicorp.com/agent-inject-secret-foo: database/roles/app
        vault.hashicorp.com/agent-inject-secret-bar: consul/creds/app
        vault.hashicorp.com/role: "app"
```
