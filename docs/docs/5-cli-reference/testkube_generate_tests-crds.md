## testkube generate tests-crds

Generate tests CRD file based on directory

### Synopsis

Generate tests manifest based on directory (e.g. for ArgoCD sync based on tests files)

```
testkube generate tests-crds <manifestDirectory> [flags]
```

### Options

```
      --env stringToString          envs in a form of name1=val1 passed to executor (default [])
      --executor-args stringArray   executor binary additional arguments
  -h, --help                        help for tests-crds
      --prerun-script string        path to script to be run before test execution
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube generate](testkube_generate.md)	 - Generate resources commands

