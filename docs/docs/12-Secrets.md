# Secret Variables
Testkube now offers many ways to pass secrets to the test and to the executors, there are following types:

### Secret Variables
Are variables passed during the test/testsuite create or run in a simple form that can be accessed in the test as an environment variable

```shell
--secret-variable <secret-name>=<secret-value>
```

Usage Example:

```shell
testkube run test my-test --secret-variable api-key="3472bjdvadncaj"
```
and it can be accessed in the test in ways that the executors allow for Postman tests it can be accessed like this:

```js
test_api_key = pm.environment.get("api-key")
```

### Secret References
If there are secrets that are already in the cluster's Secrets, Testkube offers possibility to use them as well providing secret name and the key from that secret that we want to use in the tests

```shell
--secret-variable-reference <secret-name>=<k8s-secret-name>=<secret-key>
```

Let's say that there is this secret already in the cluster

```yaml
apiVersion: v1
kind: Secret
data:
  03f9de1e-f90c-4cfa-bb64-2fc55681f1f4: eyJzZWMxIjoicGFzczEifQ==
  1cc30442-8a94-4147-96b3-25c978817d81: eyJzZWMxIjoiIn0=
  29e028e8-9fcd-408f-9ca9-babee98db13d: eyJzZWMxIjoicGFzczEifQ==
  435e41a9-dd9d-46e9-820f-142c15a7ef51: eyJzZWMxIjoiIn0=
  74c7e5f0-1032-4968-97a8-0b2b7e0b0b23: eyJzZWMxIjoiIn0=
  962fb637-3458-4c82-a430-64d169b96252: eyJzZWMxIjoicGFzczEifQ==
  current-secret: OTYyZmI2MzctMzQ1OC00YzgyLWE0MzAtNjRkMTY5Yjk2MjUy
  sec1: cGFzczE=
metadata:
  name: my-secret
  namespace: testkube
  resourceVersion: "3293159"
  uid: 8516b80b-74ba-48cf-abe6-6c9791f1b08d
type: Opaque
```

it can be used in the run or create operation as follows:

```shell
testkube run test my-test --secret-variable-reference my-sec1=my-secret=sec1
```

and in the test it can be accessed as well as an environment variable, Postman example:

```js
test_api_key = pm.environment.get("my-sec1")
```

### Secret Environment Variables

This one is for passing environment variables for the executor itself same as the `--env` but this cannot be used because it contains sensitive information, instead a k8s secret can be created and pass it with `--secret-env <k8s-secret-name>=<secret-key>`

For example if the executor needs `env-secret` from the secret

```yaml
apiVersion: v1
kind: Secret
data:
  env-secret: cGFzczE=
metadata:
  name: my-env-secret
  namespace: testkube
type: Opaque
```

it can be passed using `--secret-env my-env-secret=env-secret` and executor will receive it as `env-secret`