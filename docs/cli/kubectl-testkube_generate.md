## kubectl-testkube generate

Generate resources commands

```
kubectl-testkube generate <resourceName> [flags]
```

### Options

```
  -h, --help   help for generate
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube generate doc](kubectl-testkube_generate_doc.md)	 - Generate docs for kubectl testkube
* [kubectl-testkube generate tests-crds](kubectl-testkube_generate_tests-crds.md)	 - Generate tests CRD file based on directory

