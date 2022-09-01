## kubectl-testkube delete executor

Delete Executor

### Synopsis

Delete Executor Resource, pass name to delete by name

```
kubectl-testkube delete executor [executorName] [flags]
```

### Options

```
  -h, --help            help for executor
  -l, --label strings   label key value pair: --label key1=value1
  -n, --name string     unique executor name, you can also pass it as first argument
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --namespace string   kubernetes namespace (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            should I show additional debug messages
```

### SEE ALSO

* [kubectl-testkube delete](kubectl-testkube_delete.md)	 - Delete resources

