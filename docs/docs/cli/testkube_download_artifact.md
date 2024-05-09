## testkube download artifact

download artifact

```
testkube download artifact <executionName> <fileName> <destinationDir> [flags]
```

### Options

```
  -c, --client string         Client used for connecting to testkube API one of proxy|direct|cluster (default "proxy")
  -d, --destination string    name of the file
  -e, --execution-id string   ID of the execution
  -f, --filename string       name of the file
  -h, --help                  help for artifact
      --verbose               should I show additional debug messages
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
```

### SEE ALSO

* [testkube download](testkube_download.md)	 - Artifacts management commands

