## testkube download artifacts

download artifacts

```
testkube download artifacts <executionName> [flags]
```

### Options

```
  -c, --client string         Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --download-dir string   download dir (default "artifacts")
  -e, --execution-id string   ID of the execution
  -h, --help                  help for artifacts
      --verbose               should I show additional debug messages
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
```

### SEE ALSO

* [testkube download](testkube_download.md)	 - Artifacts management commands

