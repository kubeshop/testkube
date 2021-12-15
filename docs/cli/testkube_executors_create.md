## testkube executors create

Create new Executor

### Synopsis

Create new Executor Custom Resource

```sh
testkube executors create [flags]
```

### Options

```sh
      --executor-type string     executor type (defaults to job) (default "job")
  -h, --help                     help for create
  -i, --image string             if uri is git repository we can set additional branch parameter
  -n, --name string              unique script name - mandatory
  -t, --types stringArray        types handled by exeutor
  -u, --uri string               if resource need to be loaded from URI
      --volume-mount string      where volume should be mounted
      --volume-quantity string   how big volume should be
```

### Options inherited from parent commands

```sh
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube executors](testkube_executors.md)  - Executor management commands
