## testkube config api-uri

Set api uri for testkube client

```
testkube config api-uri <value> [flags]
```

### Options

```
  -h, --help   help for api-uri
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "https://demo.testkube.io/results")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --insecure           insecure connection for direct client
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube config](testkube_config.md)	 - Set feature configuration value

