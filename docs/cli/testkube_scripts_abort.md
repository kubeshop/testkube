## testkube scripts abort

Aborts execution of the script

```
testkube scripts abort [flags]
```

### Options

```
  -h, --help   help for abort
```

### Options inherited from parent commands

```
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube scripts](testkube_scripts.md)	 - Scripts management commands

