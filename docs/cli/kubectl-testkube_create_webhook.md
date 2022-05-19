## kubectl-testkube create webhook

Create new Webhook

### Synopsis

Create new Webhook Custom Resource

```
kubectl-testkube create webhook [flags]
```

### Options

```
  -e, --events stringArray     event types handled by executor e.g. start-test|end-test
  -h, --help                   help for webhook
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
  -n, --name string            unique webhook name - mandatory
  -u, --uri string             URI which should be called when given event occurs
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource

