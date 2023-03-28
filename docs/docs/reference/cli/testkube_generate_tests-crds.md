## testkube generate tests-crds

Generate tests CRD file based on directory

### Synopsis

Generate tests manifest on stdout, based on the source files present in a directory (e.g. for ArgoCD sync based on tests files)

* CRDs are only generated the discovered files are matching one of the built-in test types registered patterns, such as postman files which should match the pattern `.postman-collection` and end with `.json`. 
* The generated `Test` yaml spec files is templated with the followings:
   * `metadata.name` is based on the discovered file name
   * `metadata.namespace` is the configured namespace in the current kubeconfig context
   * `spec.type` is the matching adapter test type (e.g. `postman/collection`)
   * `spec.content` is inlined source file content

Check out https://github.com/kubeshop/testkube/blob/a42605cbdb84efd5e1156e2290c44b2f9b484190/pkg/test/detector/interface.go#L12-L14 for the full list of currently supported adapters 

```
testkube generate tests-crds <manifestDirectory> [flags]
```

### Options

```
      --executor-args stringArray                  executor binary additional arguments
  -h, --help                                       help for tests-crds
      --prerun-script string                       path to script to be run before test execution
      --secret-variable-reference stringToString   secret variable references in a form name1=secret_name1=secret_key1 (default [])
  -v, --variable stringToString                    variable key value pair: --variable key1=value1 (default [])
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube generate](testkube_generate.md)	 - Generate resources commands
* [kubeshop/testkube-flux](https://github.com/kubeshop/testkube-flux/blob/833f2c41861fd7191da3a465902f1c91eea5c8cc/README.md?plain=1#L79-L87) - Sample usage with flux 
