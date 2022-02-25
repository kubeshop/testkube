## testkube tests update

Update test

### Synopsis

Update Test Custom Resource

```
testkube tests update [flags]
```

### Options

```
  -f, --file string                test file - will try to read content from stdin if not specified
      --git-branch string          if uri is git repository we can set additional branch parameter
      --git-path string            if repository is big we need to define additional path to directory/file to checkout partially
      --git-token string           if git repository is private we can use token as an auth parameter
      --git-uri string             Git repository uri
      --git-username string        if git repository is private we can use username as an auth parameter
  -h, --help                       help for update
  -n, --name string                unique test name - mandatory
      --tags strings               comma separated list of tags: --tags tag1,tag2,tag3
      --test-content-type string   content type of test one of string|file-uri|git-file|git-dir
  -t, --type string                test type (defaults to postman-collection)
      --uri string                 URI of resource - will be loaded by http GET
```

### Options inherited from parent commands

```
      --analytics-enabled    should analytics be enabled (default true)
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube tests](testkube_tests.md)	 - Tests management commands

