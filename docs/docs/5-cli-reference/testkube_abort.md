## testkube abort

Abort tests or test suites

```
testkube abort <resourceName> [flags]
```

### Options

```
  -h, --help   help for abort
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
* [testkube abort execution](testkube_abort_execution.md)	 - Aborts execution of the test
* [testkube abort testsuiteexecution](testkube_abort_testsuiteexecution.md)	 - Abort test suite execution

