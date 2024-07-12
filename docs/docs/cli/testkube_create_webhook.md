## testkube create webhook

Create new Webhook

### Synopsis

Create new Webhook Custom Resource

```
testkube create webhook [flags]
```

### Options

```
      --disable                             disable webhook
  -e, --events stringArray                  event types handled by webhook e.g. start-test|end-test
      --header stringToString               webhook header value pair (golang template supported): --header Content-Type=application/xml (default [])
  -h, --help                                help for webhook
  -l, --label stringToString                label key value pair: --label key1=value1 (default [])
  -n, --name string                         unique webhook name - mandatory
      --on-state-change                     specify whether webhook should be triggered only on a state change
      --payload-field string                field to use for notification object payload
      --payload-template string             if webhook needs to send a custom notification, then a path to template file should be provided
      --payload-template-reference string   reference to payload template to use for the webhook
      --selector string                     expression to select tests and test suites for webhook events: --selector app=backend
      --update                              update, if webhook already exists
  -u, --uri string                          URI which should be called when given event occurs (golang template supported)
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --crd-only           generate only crd
      --insecure           insecure connection for direct client
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource

