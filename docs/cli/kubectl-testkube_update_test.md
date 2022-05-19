## kubectl-testkube update test

Update test

### Synopsis

Update Test Custom Resource

```
kubectl-testkube update test [flags]
```

### Options

```
  -f, --file string                      test file - will try to read content from stdin if not specified
      --git-branch string                if uri is git repository we can set additional branch parameter
      --git-path string                  if repository is big we need to define additional path to directory/file to checkout partially
      --git-token string                 if git repository is private we can use token as an auth parameter
      --git-uri string                   Git repository uri
      --git-username string              if git repository is private we can use username as an auth parameter
  -h, --help                             help for test
  -l, --label stringToString             label key value pair: --label key1=value1 (default [])
  -n, --name string                      unique test name - mandatory
      --schedule string                  test schedule in a cronjob form: * * * * *
  -s, --secret-variable stringToString   secret variable key value pair: -s key1=value1 (default [])
      --test-content-type string         content type of test one of string|file-uri|git-file|git-dir
  -t, --type string                      test type (defaults to postman-collection)
      --uri string                       URI of resource - will be loaded by http GET
  -v, --variable stringToString          variable key value pair: -v key1=value1 (default [])
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set (default "http://testdash.testkube.io/api")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube update](kubectl-testkube_update.md)	 - Update resource

