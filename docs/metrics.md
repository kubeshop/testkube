# Metrics

The Kubtest API Server exposes a `/metrics` endpoint that can be consumed by Prometheus, Grafana, etc. It
currently exposes the following metrics:

* `kubtest_executions_count` - The total number of script executions
* `kubtest_scripts_creation_count` - The total number of scripts created by type events
* `kubtest_scripts_abort_count` - The total number of scripts created by type events


## Installation

If yout don't have installed prometheus operator please follow https://grafana.com/docs/grafana-cloud/quickstart/prometheus_operator/ first 

Next you'll need to add `ServiceMonitor` custom resource to your cluster which will scrape metrics from our
kubtest API server.

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kubtest-api-server
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

If you're installing kubtest manually by our Helm chart you can pass `prometheus.enabled` value to install 
command: 



## Grafana dashboard 

If you want use our dashboard please import this json definition:

https://raw.githubusercontent.com/kubeshop/kubtest/main/assets/grafana-dasboard.json
