## testkube watch

Watch tests or test suites

```
testkube watch <resourceName> [flags]
```

### Options

```
  -h, --help   help for watch
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube watch execution](testkube_watch_execution.md)	 - Watch logs output from executor pod
* [testkube watch testsuiteexecution](testkube_watch_testsuiteexecution.md)	 - Watch test suite

