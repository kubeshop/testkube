# Test CRD Generation

## File Naming Convention for Test CRD Generation

We support the following file naming convention for Postman files to support multiple environment files:

### Test Filename Convention

`<Test name>.postman_collection.json` - Where `Test name` should be reused for the environment files.

For example, mytest.postman_collection.json.

### Test Environment File Naming Convention

`<Test name>.<Test env>.postman_environment.json` - Where `Test name` is reused from the test files and 
`Test env` is pointing to a particular testing environment.

For example, mytest.prod.postman_collection.json.

### Test Secret Environment File Naming Convention

`<Test name>.<Test env>.postman_secret_environment.json` - Where `Test name` is reused from test files and 
`Test env` is pointing to particular testing environment.

For example, mytest.prod.postman_secret_environment.json.

It is expected that each variable value in a secret environment file is provided in the form of `secret-name=secret-key`.
In this case, it will be added to a list of Test secret variables.
For example,

```json
{
        "id": "f8a038bf-3766-4424-94ee-381a69f55b9a",
        "name": "Testing secret env",
        "values": [
                {
                        "key": "secvar1",
                        "value": "var-secrets=homepage",
                        "enabled": true
                },
                {
                        "key": "secvar2",
                        "value": "var-secrets=apikey",
                        "enabled": false
                }
        ],
        "_postman_variable_scope": "environment",
        "_postman_exported_at": "2022-09-04T04:47:42.590Z",
        "_postman_exported_using": "Postman/9.14.14"
}
```

will add this section to Test CRD (only secvar1, because secvar2 is disabled):

```yaml
  executionRequest:
    variables:
    - name: secvar1
      type: secret
      secretRef:
        name: var-secrets
        key: homepage
```