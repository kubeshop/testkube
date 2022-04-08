## kubectl-testkube delete executor

Delete Executor

### Synopsis

Delete Executor Resource, pass name to delete by name

```
kubectl-testkube delete executor [executorName] [flags]
```

### Options

```
  -h, --help          help for executor
  -n, --name string   unique executor name, you can also pass it as first argument
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [kubectl-testkube delete](kubectl-testkube_delete.md)	 - Delete resources

