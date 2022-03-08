## testkube create webhook

Create new Webhook

### Synopsis

Create new Webhook Custom Resource

```
testkube create webhook [flags]
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
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource

