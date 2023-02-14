## testkube create testsource

Create new TestSource

### Synopsis

Create new TestSource Custom Resource

```
testkube create testsource [flags]
```

### Options

```
  -f, --file string                          source file - will be read from stdin if not specified
      --git-branch string                    if uri is git repository we can set additional branch parameter
      --git-certificate-secret string        if git repository is private we can use certificate as an auth parameter stored in a kubernetes secret name
      --git-commit string                    if uri is git repository we can use commit id (sha) parameter
      --git-path string                      if repository is big we need to define additional path to directory/file to checkout partially
      --git-token string                     if git repository is private we can use token as an auth parameter
      --git-token-secret stringToString      git token secret in a form of secret_name1=secret_key1 for private repository (default [])
      --git-uri string                       Git repository uri
      --git-username string                  if git repository is private we can use username as an auth parameter
      --git-username-secret stringToString   git username secret in a form of secret_name1=secret_key1 for private repository (default [])
      --git-working-dir string               if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter
  -h, --help                                 help for testsource
  -l, --label stringToString                 label key value pair: --label key1=value1 (default [])
  -n, --name string                          unique test source name - mandatory
      --source-type string                   source type of test one of string|file-uri|git-file|git-dir
  -u, --uri string                           URI which should be called when given event occurs
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --crd-only           generate only crd
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource

