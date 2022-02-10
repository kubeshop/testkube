## testkube scripts list

Get all available scripts

### Synopsis

Getting all available scritps from given namespace - if no namespace given "testkube" namespace is used

```sh
testkube scripts list [flags]
```

### Options

```sh
  -h, --help   help for list
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

* [testkube scripts](testkube_scripts.md)  - Tests management commands
