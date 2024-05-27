# Compatibility with Istio

TODO add a primer on Istio and the various possible configurations

## Disable Istio for test executions of prebuilt/container executors

Chart `testkube`

```yaml
testkube-api:
    jobPodAnnotations:
        sidecar.istio.io/inject: "false"
```

## Disable Istio for agent hooks

Chart `testkube`

```yaml
preUpgradeHook:
    podAnnotations:
        sidecar.istio.io/inject: "false"
preUpgradeHookNATS:
    podAnnotations:
        sidecar.istio.io/inject: "false"
```

## Disable Istio for operator hooks

Chart `testkube`

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

## Define a global test workflows template

Chart `testkube`

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

## Hold the API server till Istio's proxy is ready

This should avoid the issues with the enterprise API pods failing on restart.

Chart `testkube-enterprise`

```yaml
testkube-cloud-api:
    podAnnotations:
        proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
```

## Hold worker service till Istio's proxy is ready

This should avoid the issues with the worker service pods failing on restart.

Chart `testkube-enterprise`

```yaml
testkube-worker-service:
    podAnnotations:
        proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
```
