## testkube set context

Set context data for Testkube Cloud

```
testkube set context <value> [flags]
```

### Options

```
  -k, --api-key string           API Key for Testkube Cloud
      --cloud-agent-key string   Agent Key for Testkube Cloud
      --cloud-agent-uri string   Testkube Cloud Agent URI (default "agent.testkube.io:443")
      --cloud-api-uri string     Testkube Cloud API URI (default "https://api.testkube.io")
  -e, --env string               Testkube Cloud environment ID
  -h, --help                     help for context
      --kubeconfig               reset context mode for CLI to default kubeconfig based
  -o, --org string               Testkube Cloud organization ID
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube set](testkube_set.md)	 - Set resources

