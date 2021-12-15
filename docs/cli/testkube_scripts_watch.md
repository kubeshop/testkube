## testkube scripts watch

Watch until script execution is in complete state

### Synopsis

Gets script execution details, until it's in success/error state, blocks until gets complete state

```sh
testkube scripts watch [flags]
```

### Options

```sh
  -h, --help   help for watch
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

* [testkube scripts](testkube_scripts.md)  - Scripts management commands
