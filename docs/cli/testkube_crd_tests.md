## testkube crd tests

Generate tests CRD file based on directory

### Synopsis

Generate tests manifest based on directory (e.g. for ArgoCD sync based on tests files)

```
testkube crd tests <manifestDirectory> [flags]
```

### Options

```
  -h, --help   help for tests
```

### Options inherited from parent commands

```
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube crd](testkube_crd.md)	 - CRDs management commands

