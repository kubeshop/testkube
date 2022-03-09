# Testkube Tests Create

Create a new test.

## **Synopsis**

Create a new Test Custom Resource:

```
testkube create test [flags]
```

## **Options**

```
  -f, --file string                Test file - will be read from stdin if not specified.
      --git-branch string          If URI is Git repository, we can set an additional branch parameter.
      --git-path string            If the repository is big, we need to define additional path to directory/file to checkout partially.
      --git-token string           If Git repository is private, we can use a token as an auth parameter.
      --git-uri string             Git repository URI.
      --git-username string        If git repository is private, we can use username as an auth parameter.
  -h, --help                       Help for create.
  -n, --name string                Unique test name - mandatory.
      --tags strings               Comma separated list of tags: --tags tag1,tag2,tag3.
      --test-content-type string   Content type of test. Either string, file-uri, git-file or git-dir.
  -t, --type string                Test type (defaults to postman/collection).
      --uri string                 URI of resource - will be loaded by http GET.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled    Enable analytics be (default "true").
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy").
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -s, --namespace string     Kubernetes namespace (default "testkube").
  -o, --output string        Output type - raw, json or go  (default "raw").
  -v, --verbose              Show additional debug messages.
```

## **SEE ALSO**

* [Testkube Tests](testkube_tests.md)	 - Tests management commands.

