## kubtest scripts list

Get all available scripts

### Synopsis

Getting all available scritps from given namespace - if no namespace given "default" namespace is used

```
kubtest scripts list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "default")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [kubtest scripts](kubtest_scripts.md)	 - Scripts management commands

