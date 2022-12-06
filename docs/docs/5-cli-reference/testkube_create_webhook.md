## testkube create webhook

Create new Webhook

### Synopsis

Create new Webhook Custom Resource

```
testkube create webhook [flags]
```

### Options

```
  -e, --events stringArray     event types handled by executor e.g. start-test|end-test
  -h, --help                   help for webhook
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
  -n, --name string            unique webhook name - mandatory
      --selector string        expression to select tests and test suites for webhook events: --selector app=backend
  -u, --uri string             URI which should be called when given event occurs
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --crd-only           generate only crd
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource

