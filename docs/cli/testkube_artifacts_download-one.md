# Testkube Download Single Artifact

Download artifact:

```
testkube artifacts download-one <executionID> <fileName> <destinationDir> [flags]
```

## **Options**

```
  -d, --destination string    Name of the file.
  -e, --execution-id string   ID of the execution.
  -f, --filename string       Name of the file.
  -h, --help                  Help for download-one.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled   Enable analytics (default "true").
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy").
  -s, --namespace string    Kubernetes namespace (default "testkube").
  -v, --verbose             Show additional debug messages.
```

## **SEE ALSO**

* [Testkube Artifacts](testkube_artifacts.md)	 - Artifacts management commands.

