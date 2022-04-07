## kubectl-testkube create webhook

Create new Webhook

### Synopsis

Create new Webhook Custom Resource

```
kubectl-testkube create webhook [flags]
```

### Options

```
  -e, --events stringArray   event types handled by executor e.g. start-test|end-test
  -h, --help                 help for webhook
  -n, --name string          unique webhook name - mandatory
  -u, --uri string           URI which should be called when given event occurs
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
  -v, --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource

