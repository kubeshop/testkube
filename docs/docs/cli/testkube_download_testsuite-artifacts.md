## testkube download testsuite-artifacts

download test suite artifacts

```
testkube download testsuite-artifacts <executionName> [flags]
```

### Options

```
  -c, --client string         Client used for connecting to testkube API one of proxy|direct|cluster (default "proxy")
      --download-dir string   download dir (default "artifacts")
  -e, --execution-id string   ID of the test suite execution
      --format string         data format for storing files, one of folder|archive (default "folder")
  -h, --help                  help for testsuite-artifacts
      --mask stringArray      regexp to filter downloaded files, single or comma separated, like report/.* or .*\.json,.*\.js$
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

