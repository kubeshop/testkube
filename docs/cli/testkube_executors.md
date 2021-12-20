## testkube executors

Executor management commands

### Synopsis

All available scripts and scripts executions commands

```sh
testkube executors [flags]
```

### Options

```sh
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -h, --help                 help for executors
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
* [testkube executors create](testkube_executors_create.md)  - Create new Executor
* [testkube executors delete](testkube_executors_delete.md)  - Gets executordetails
* [testkube executors get](testkube_executors_get.md)  - Gets executordetails
* [testkube executors list](testkube_executors_list.md)  - Gets executors
