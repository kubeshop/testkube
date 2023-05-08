# Secret Variables

Testkube now offers many ways to pass secrets to the test and to the executors. Here are following types:

## Secret Variables

Secret Variables are variables passed during the test/test suite creation or run in a simple form that can be accessed in the test as an environment variable.

```sh
--secret-variable <secret-name>=<secret-value>
```

Usage Example:

```sh
testkube run test my-test --secret-variable api-key="3472bjdvadncaj"
```

It can be accessed in the test in ways that the executors allow. For Postman tests it can be accessed like this:

```js
test_api_key = pm.environment.get("api-key");
```

### Secret References

If there are secrets that are already in the cluster's Secrets, Testkube allows you to use them and provides the secret name and the key from that secret that will be used in the tests.

```sh
--secret-variable-reference <secret-name>=<k8s-secret-name>=<secret-key>
```

Let's say that this secret already in the cluster:

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

This secret can be used in the run or create operation as follows:

```sh
testkube run test my-test --secret-variable-reference my-sec1=my-secret=sec1
```

And, in the test, it can be accessed as an environment variable. Here is a Postman example:

```js
test_api_key = pm.environment.get("my-sec1");
```

### Secret Environment Variables

Secret Environment Variables pass environment variables for the executor itself like `--env` when `--env` cannot be used because it contains sensitive information. Instead a k8s secret can be created and passed with `--secret-env <secret-key>=<k8s-secret-name>`.

For example, if the executor needs `env-secret` from the secret,
it can be passed using `--secret-env env-secret=my-env-secret` and executor will receive it as `env-secret`:

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

:::info
Postman expects the JSON object in the secret with the postman env file data json format:

```js
{
	"id": "72a6e5d0-933d-4b64-84df-6de2f441be08",
	"name": "env name",
	"values": [
		{
			"key": "VAR_NAME",
			"value": "https://example.com",
			"enabled": true
		}
  ],
	"_postman_variable_scope": "environment",
	"_postman_exported_at": "2022-06-06T13:35:46.820Z",
	"_postman_exported_using": "Postman/9.21.1"
}
```
:::
