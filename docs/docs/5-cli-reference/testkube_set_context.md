## testkube set context

Set context data for Testkube Cloud

```
testkube set context <value> [flags]
```

### Options

```
  -k, --api-key string         API Key for Testkube Cloud
      --cloud-api-uri string   https://api.testkube.io
  -e, --env string             Testkube Cloud environment ID
  -h, --help                   help for context
      --kubeconfig             reset context mode for CLI to default kubeconfig based
  -o, --org string             Testkube Cloud organization ID
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube set](testkube_set.md)	 - Set resources

