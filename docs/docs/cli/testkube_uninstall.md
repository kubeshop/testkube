# testkube uninstall

Uninstall Helm chart registry in current kubectl context

### Synopsis

Uninstall Helm chart registry in current kubectl context

```
testkube uninstall [flags]
```

### Options

```
  -h, --help          help for uninstall
      --name string   installation name (default "testkube")
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data
      --verbose             show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin

