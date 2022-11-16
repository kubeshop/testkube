## testkube watch testsuiteexecution

Watch test

### Synopsis

Watch test by test execution ID, returns results to console

```
testkube watch testsuiteexecution <executionID> [flags]
```

### Options

```
  -h, --help   help for testsuiteexecution
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

* [testkube watch](testkube_watch.md)	 - Watch tests or test suites

