## testkube scripts create

Create new test

### Synopsis

Create new Script Custom Resource,

```sh
testkube scripts create [flags]
```

### Options

```sh
  -f, --file string         test file - will be read from stdin if not specified
      --git-branch string   if uri is git repository we can set additional branch parameter
      --git-path string     if repository is big we need to define additional path to directory/file to checkout partially
  -h, --help                help for create
  -n, --name string         unique test name - mandatory
  -t, --type string         test type (defaults to postman-collection)
      --uri string          if resource need to be loaded from URI
```

### Options inherited from parent commands

```sh
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube tests](testkube_tests.md)  - Tests management commands
