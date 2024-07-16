## testkube create testworkflow

Create test workflow

```
testkube create testworkflow [flags]
```

### Options

```
  -f, --file string   file path to get the test workflow specification
  -h, --help          help for testworkflow
      --name string   test workflow name
      --update        update, if test workflow already exists
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --crd-only                generate only crd
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource

