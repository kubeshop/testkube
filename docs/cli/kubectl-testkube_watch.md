## kubectl-testkube watch

Watch tests or test suites

```
kubectl-testkube watch <resourceName> [flags]
```

### Options

```
  -h, --help   help for watch
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
  -v, --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube watch execution](kubectl-testkube_watch_execution.md)	 - Watch logs output from executor pod
* [kubectl-testkube watch testsuiteexecution](kubectl-testkube_watch_testsuiteexecution.md)	 - Watch test

