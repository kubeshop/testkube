## testkube artifacts download-one

download artifact

```
testkube artifacts download-one <executionID> <fileName> <destinationDir> [flags]
```

### Options

```
  -d, --destination string    name of the file
  -e, --execution-id string   ID of the execution
  -f, --filename string       name of the file
  -h, --help                  help for download-one
```

### Options inherited from parent commands

```
      --analytics-enabled   should analytics be enabled (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [testkube artifacts](testkube_artifacts.md)	 - Artifacts management commands

