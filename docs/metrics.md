# Metrics

The TestKube API Server exposes a `/metrics` endpoint that can be consumed by Prometheus, Grafana, etc. It
currently exposes the following metrics:

* `testkube_executions_count` - The total number of script executions
* `testkube_scripts_creation_count` - The total number of scripts created by type events
* `testkube_scripts_abort_count` - The total number of scripts created by type events


## Installation

If yout don't have installed Prometheus operator please follow [https://grafana.com/docs/grafana-cloud/quickstart/prometheus_operator/](https://grafana.com/docs/grafana-cloud/quickstart/prometheus_operator/) first 

Next you'll need to add `ServiceMonitor` custom resource to your cluster which will scrape metrics from our
TestKube API server.

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

If you're installing TestKube manually by our Helm chart you can pass `prometheus.enabled` value to install 
command: 



## Grafana dashboard 

If you want use our dashboard please import this json definition:

[https://github.com/kubeshop/testkube/blob/main/assets/grafana-dasboard.json](https://github.com/kubeshop/testkube/blob/main/assets/grafana-dasboard.json)
