## testkube executors delete

Gets executordetails

### Synopsis

Gets executor, you can change output format

```
testkube executors delete [flags]
```

### Options

```
  -h, --help          help for delete
  -n, --name string   unique executor name, you can also pass it as first argument
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

* [testkube executors](testkube_executors.md)	 - Executor management commands

