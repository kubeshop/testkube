# Using Istio

An Istio mesh can be configured in several ways.

Traffic can be routed to the proxy by using either the `istio-init` container,
the [CNI plugin](https://istio.io/latest/docs/setup/additional-setup/cni/), or
[ambient mode](https://istio.io/latest/blog/2022/introducing-ambient-mesh/)
(currently in [beta](https://istio.io/latest/blog/2024/ambient-reaches-beta/)).
In ambient mode, the proxy can run as a separate pod, but in the other modes, it
runs as a sidecar container.

If using Kubernetes 1.28+ with the [`SidecarContainers` feature
gate](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/)
enabled and Istio 1.19+, it is highly recommended that Istio be configured to
use native sidecars. Without native sidecars, Istio has several issues which can
be put in the following buckets:

-   Networking from within other init containers either bypasses the proxy by
    default (non-CNI) or [should be
    configured](https://istio.io/latest/docs/setup/additional-setup/cni/#compatibility-with-application-init-containers)
    to bypass the proxy (CNI plugin).
-   Containers [must wait for the proxy to be
    ready](https://github.com/istio/istio/issues/11130), otherwise, network requests
    may fail.
-   Containers, [especially within batch
    jobs](https://github.com/istio/istio/issues/6324), must signal to the proxy
    that they have completed, otherwise, the proxy never exits and the job never
    completes.

## Compatibility with Istio

Testkube has been verified to work with Istio installations utilizing native
sidecars. Compatibility with ambient mode or the CNI plugin has not been
verified, but we are ready to support enterprise customers using these
particular configurations.

Installations that do not currently support native sidecars should apply the
configurations below to utilize Testkube until they can upgrade their cluster to
enable native sidecars:

### Configurations for Istio installations without native sidecars

All the values mentioned will be from the top of the specified chart.

#### Disable Istio for test executions of prebuilt/container executors

Chart `testkube`:

```yaml
testkube-api:
    jobPodAnnotations:
        sidecar.istio.io/inject: "false"
```

#### Disable Istio for agent hooks

Chart `testkube`:

```yaml
preUpgradeHook:
    podAnnotations:
        sidecar.istio.io/inject: "false"
preUpgradeHookNATS:
    podAnnotations:
        sidecar.istio.io/inject: "false"
```

#### Disable Istio for operator hooks

Chart `testkube`:

```yaml
testkube-operator:
    preUpgrade:
        podAnnotations:
            sidecar.istio.io/inject: "false"
    webhook:
        patch:
            podAnnotations:
                sidecar.istio.io/inject: "false"
```

#### Define a global test workflows template

Chart `testkube`:

```yaml
global:
    testWorkflows:
        globalTemplate:
            enabled: true
            spec:
                pod:
                    annotations:
                        sidecar.istio.io/inject: "false"
```

#### Hold the API server till Istio's proxy is ready

This should avoid the issues with the enterprise API pods failing on restart.

Chart `testkube-enterprise`:

```yaml
testkube-cloud-api:
    podAnnotations:
        proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
```

#### Hold worker service till Istio's proxy is ready

This should avoid the issues with the worker service pods failing on restart.

Chart `testkube-enterprise`:

```yaml
testkube-worker-service:
    podAnnotations:
        proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
```
