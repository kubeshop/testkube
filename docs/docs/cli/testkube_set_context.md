## testkube set context

Set context data for Testkube Cloud

```
testkube set context <value> [flags]
```

### Options

```
  -k, --api-key string             API Key for Testkube Cloud
      --cloud-root-domain string   defaults to testkube.io, usually you don't need to change it (default "testkube.io")
  -e, --env string                 Testkube Cloud Environment ID
  -h, --help                       help for context
      --kubeconfig                 reset context mode for CLI to default kubeconfig based
  -n, --namespace string           Testkube namespace to use for CLI commands
  -o, --org string                 Testkube Cloud Organization ID
```

### Options inherited from parent commands

```
  -a, --api-uri string   api uri, default value read from config if set (default "https://demo.testkube.io/results/v1")
  -c, --client string    client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --oauth-enabled    enable oauth
      --verbose          show additional debug messages
```

### SEE ALSO

* [testkube set](testkube_set.md)	 - Set resources

