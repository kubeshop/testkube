## kubectl-testkube config

Set feature configuration value

```
kubectl-testkube config <feature> <value> [flags]
```

### Options

```
  -h, --help   help for config
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

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube config api-uri](kubectl-testkube_config_api-uri.md)	 - Set api uri for testkube client
* [kubectl-testkube config namespace](kubectl-testkube_config_namespace.md)	 - Set namespace for testkube client
* [kubectl-testkube config oauth](kubectl-testkube_config_oauth.md)	 - Set oauth credentials for api uri in testkube client

