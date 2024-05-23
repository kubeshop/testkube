## testkube delete

Delete resources

```
testkube delete <resourceName> [flags]
```

### Options

```
  -c, --client string   Client used for connecting to testkube API one of proxy|direct|cluster (default "proxy")
  -h, --help            help for delete
      --verbose         should I show additional debug messages
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube delete executor](testkube_delete_executor.md)	 - Delete Executor
* [testkube delete template](testkube_delete_template.md)	 - Delete a template.
* [testkube delete test](testkube_delete_test.md)	 - Delete Test
* [testkube delete testsource](testkube_delete_testsource.md)	 - Delete test source
* [testkube delete testsuite](testkube_delete_testsuite.md)	 - Delete test suite
* [testkube delete testworkflow](testkube_delete_testworkflow.md)	 - Delete test workflows
* [testkube delete testworkflowtemplate](testkube_delete_testworkflowtemplate.md)	 - Delete test workflow templates
* [testkube delete webhook](testkube_delete_webhook.md)	 - Delete webhook

