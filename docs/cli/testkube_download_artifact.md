## testkube download artifact

download artifact

```
testkube download artifact <executionID> <fileName> <destinationDir> [flags]
```

### Options

```
  -d, --destination string    name of the file
  -e, --execution-id string   ID of the execution
  -f, --filename string       name of the file
  -h, --help                  help for artifact
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [testkube download](testkube_download.md)	 - Artifacts management commands

