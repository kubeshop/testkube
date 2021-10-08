## testkube scripts create

Create new script

### Synopsis

Create new Script Custom Resource, 

```
testkube scripts create [flags]
```

### Options

```
  -f, --file string         script file - will be read from stdin if not specified
      --git-branch string   if uri is git repository we can set additional branch parameter
      --git-path string     if repository is big we need to define additional path to directory/file to checkout partially
  -h, --help                help for create
  -n, --name string         unique script name - mandatory
  -t, --type string         script type (defaults to postman-collection) (default "postman/collection")
      --uri string          if resource need to be loaded from URI
```

### Options inherited from parent commands

```
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "default")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube scripts](testkube_scripts.md)	 - Scripts management commands

