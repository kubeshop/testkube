## testkube run

Runs tests, test suites or test workflows

```
testkube run <resourceName> [flags]
```

### Options

```
  -h, --help   help for run
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube run test](testkube_run_test.md)	 - Starts new test
* [testkube run testsuite](testkube_run_testsuite.md)	 - Starts new test suite
* [testkube run testworkflow](testkube_run_testworkflow.md)	 - Starts test workflow execution

