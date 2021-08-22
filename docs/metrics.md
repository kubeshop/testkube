# Metrics

The Kubtest API Server exposes a `/metrics` endpoint that can be consumed by Prometheus, Grafana, etc. It
currently exposes the following metrics:

* `kubtest_executions_count` - The total number of script executions
* `kubtest_scripts_creation_count` - The total number of scripts created by type events
* `kubtest_scripts_abort_count` - The total number of scripts created by type events


