## testkube delete executor

Delete Executor

### Synopsis

Delete Executor Resource, pass name to delete by name

```
testkube delete executor [executorName] [flags]
```

### Options

```
  -h, --help          help for executor
  -n, --name string   unique executor name, you can also pass it as first argument
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [testkube delete](testkube_delete.md)	 - Delete resources

