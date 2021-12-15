## testkube install

Install Helm chart registry in current kubectl context

### Synopsis

Install can be configured with use of particular

```sh
testkube install [flags]
```

### Options

```sh
      --chart string   chart name (default "kubeshop/testkube")
  -h, --help           help for install
      --name string    installation name (default "testkube")
```

### Options inherited from parent commands

```sh
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)  - testkube entrypoint for plugin
