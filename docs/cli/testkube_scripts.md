## testkube scripts

Tests management commands

### Synopsis

All available scripts and scripts executions commands

```sh
testkube scripts [flags]
```

### Options

```sh
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -h, --help                 help for scripts
  -o, --output string        output type one of raw|json|go  (default "raw")
```

### Options inherited from parent commands

```sh
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)  - testkube entrypoint for plugin
* [testkube scripts abort](testkube_scripts_abort.md)  - Aborts execution of the test
* [testkube scripts create](testkube_scripts_create.md)  - Create new test
* [testkube scripts execution](testkube_scripts_execution.md)  - Gets test execution details
* [testkube scripts executions](testkube_scripts_executions.md)  - List scripts executions
* [testkube scripts list](testkube_scripts_list.md)  - Get all available scripts
* [testkube scripts start](testkube_scripts_start.md)  - Starts new test
* [testkube scripts watch](testkube_scripts_watch.md)  - Watch until test execution is in complete state
