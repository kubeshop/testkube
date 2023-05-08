# testkube debug create-ticket

Create bug ticket

### Synopsis

Create an issue of type bug in the Testkube repository

```
testkube debug create-ticket [flags]
```

### Options

```
  -h, --help                   help for info
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

* [testkube](testkube.md)  - Testkube entrypoint for kubectl plugin
* [testkube debug](testkube_debug.md) - Debug Testkube
