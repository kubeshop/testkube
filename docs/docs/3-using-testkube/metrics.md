---
sidebar_position: 8
sidebar_label: Metrics
---
# Metrics

The Testkube API Server exposes a `/metrics` endpoint that can be consumed by Prometheus, Grafana, etc. Currently, the following metrics are exposed:

* `testkube_test_executions_count` - The total number of test executions.
* `testkube_testsuite_executions_count` - The total number of test suite executions.
* `testkube_test_creations_count` - The total number of tests created by type events.
* `testkube_testsuite_creations_count` - The total number of test suites created events.
* `testkube_test_updates_count` - The total number of tests updated by type events.
* `testkube_testsuite_updates_count` - The total number of test suites updated events.
* `testkube_test_aborts_count` - The total number of tests aborted by type events.

## **Installation**

If a Prometheus operator is not installed, please follow the steps here: [https://grafana.com/docs/grafana-cloud/quickstart/prometheus_operator/](https://grafana.com/docs/grafana-cloud/quickstart/prometheus_operator/).

Then, add a `ServiceMonitor` custom resource to your cluster which will scrape metrics from our
Testkube API server:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: testkube-api-server
  labels:
    app: prometheus
spec:
  endpoints:
  - interval: 10s
    port: http
  selector:
    matchLabels:
      app.kubernetes.io/name: api-server
```

If you're installing Testkube manually with our Helm chart, you can pass the `prometheus.enabled` value to the install command.

## **Grafana Dashboard**

To use the Grafana dashboard, import this JSON definition:

[https://github.com/kubeshop/testkube/blob/main/assets/grafana-dasboard.json](https://github.com/kubeshop/testkube/blob/main/assets/grafana-dasboard.json)
