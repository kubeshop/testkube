# Testkube Tests Update

## **Synopsis**

Update Test Custom Resource:

```
testkube tests update [flags]
```

## **Options**

```
  -f, --file string                Test file - will try to read content from stdin if not specified.
      --git-branch string          If URI is a Git repository, we can set additional branch parameters.
      --git-path string            If the repository is large, we need to define an additional path to the directory/file to checkout partially.
      --git-token string           If Git repository is private, use the token as an auth parameter.
      --git-uri string             Git repository URI.
      --git-username string        If Git repository is private, use the username as an auth parameter.
  -h, --help                       Help for update.
  -n, --name string                Unique test name - mandatory.
      --tags strings               A comma separated list of tags: --tags tag1,tag2,tag3.
      --test-content-type string   Content type of test one of string|file-uri|git-file|git-dir.
  -t, --type string                Test type (defaults to postman-collection).
      --uri string                 URI of resource - will be loaded by http GET.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled    Enable analytics (default "true").
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy").
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -s, --namespace string     Kubernetes namespace (default "testkube").
  -o, --output string        Output type - raw, json or go  (default "raw").
  -v, --verbose              Show additional debug messages.
```

## **SEE ALSO**

* [Testkube Tests](testkube_tests.md)	 - Tests management commands.

