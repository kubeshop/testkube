## kubtest scripts

Scripts management commands

### Synopsis

All available scripts and scripts executions commands

```
kubtest scripts [flags]
```

### Options

```
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -h, --help                 help for scripts
  -s, --namespace string     kubernetes namespace (default "default")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [kubtest](kubtest.md)	 - kubtest entrypoint for plugin
* [kubtest scripts abort](kubtest_scripts_abort.md)	 - (NOT IMPLEMENTED) Aborts execution of the script
* [kubtest scripts create](kubtest_scripts_create.md)	 - Create new script
* [kubtest scripts execution](kubtest_scripts_execution.md)	 - Gets script execution details
* [kubtest scripts executions](kubtest_scripts_executions.md)	 - List scripts executions
* [kubtest scripts list](kubtest_scripts_list.md)	 - Get all available scripts
* [kubtest scripts start](kubtest_scripts_start.md)	 - Starts new script
* [kubtest scripts watch](kubtest_scripts_watch.md)	 - Watch until script execution is in complete state

