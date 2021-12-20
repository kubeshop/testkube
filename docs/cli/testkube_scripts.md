## testkube scripts

Scripts management commands

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
* [testkube scripts abort](testkube_scripts_abort.md)  - Aborts execution of the script
* [testkube scripts create](testkube_scripts_create.md)  - Create new script
* [testkube scripts execution](testkube_scripts_execution.md)  - Gets script execution details
* [testkube scripts executions](testkube_scripts_executions.md)  - List scripts executions
* [testkube scripts list](testkube_scripts_list.md)  - Get all available scripts
* [testkube scripts start](testkube_scripts_start.md)  - Starts new script
* [testkube scripts watch](testkube_scripts_watch.md)  - Watch until script execution is in complete state
