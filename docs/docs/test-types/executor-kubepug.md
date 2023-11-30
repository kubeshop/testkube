# KubePug

[KubePug](https://github.com/kubepug/kubepug) is a kubectl plugin checking for deprecated Kubernetes clusters or deprecated versions of Kubernetes manifests. It can connect to both your cluster directly and it can run on input files.
For security, Testkube only supports scanning input files via the KubePug executor.

* Default command for this executor: `kubepug`
* Default arguments for this executor command: `--format=json` `--input-file` `<runPath>`

Parameters in `<>` are calculated at test execution:

* `<runPath>` - location of test files

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

Running the KubePug Testkube executor does not require any special installation; Testkube comes with the ability to run Kubepug immediately after installation.

## Testing Manifests

By default, `kubepug` downloads the latest `swagger.json` from the `Kubernetes` GitHub repository. When running it using Testkube, the same behavior is applied, unless a version is specified in the arguments.

Example input file:

```yaml
apiVersion: v1
conditions:
- message: '{"health":"true"}'
  status: "True"
  type: Healthy
kind: ComponentStatus
metadata:
  creationTimestamp: null
  name: etcd-1
```

To test this using Testkube, first create a test, then run it:

```bash
$ kubectl testkube create test --file file_name.yaml --type kubepug/yaml --name kubepug-example-test-1
Test created testkube / kubepug-example-test-1 🥇


$ kubectl testkube run test kubepug-example-test-1
Type          : kubepug/yaml
Name          : kubepug-example-test-1
Execution ID  : 62b59ae1657713ea1b003a25
Execution name: completely-helped-fowl



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 62b59ae1657713ea1b003a25


Use following command to get test execution details:
$ kubectl testkube get execution 62b59ae1657713ea1b003a25


$ kubectl testkube get execution 62b59ae1657713ea1b003a25
ID:        62b59ae1657713ea1b003a25
Name:      completely-helped-fowl
Type:      kubepug/yaml
Duration:  00:00:05

Status test execution failed:

⨯
{"DeprecatedAPIs":[{"Description":"ComponentStatus (and ComponentStatusList) holds the cluster validation info. Deprecated: This API is deprecated in v1.19+","Group":"","Kind":"ComponentStatus","Version":"v1","Name":"","Deprecated":true,"Items":[{"Scope":"OBJECT","ObjectName":"etcd-1","Namespace":"","location":"/tmp/test-content4075001618"}]}],"DeletedAPIs":null}
```

These tests also support input strings, file URIs, Git files and Git directories.

## Testing the Output of `kubectl get`

Another way to test Kubernetes objects is to create the Testkube Test with the output of `kubectl get`. The output has to be in the correct format in order for KubePug to be able to scan it using the `-o yaml` argument.

```bash
$ kubectl get PodSecurityPolicy gce.gke-metrics-agent -o yaml | kubectl testkube create test --type kubepug/yaml --name kubepug-example-test2
Warning: policy/v1beta1 PodSecurityPolicy is deprecated in v1.21+, unavailable in v1.25+
Test created testkube / kubepug-example-test2 🥇

$ kubectl testkube run test kubepug-example-test2
Type          : kubepug/yaml
Name          : kubepug-example-test2
Execution ID  : 62c8110338a672dc415ce98e
Execution name: mostly-rapid-lark


Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 62c8110338a672dc415ce98e

Use following command to get test execution details:
$ kubectl testkube get execution 62c8110338a672dc415ce98e
```

## Testing Against a Previous Kubernetes Version

It is possible to run the same Testkube KubePug test using different Kubernetes versions to compare to using the `--k8s-version=${VERSION}` argument as shown below:

```bash
$ kubectl testkube run test kubepug-example-test-1 --args '--k8s-version=v1.18.0'
Type          : kubepug/yaml
Name          : kubepug-example-test-1
Execution ID  : 62b59d52657713ea1b003a2d
Execution name: notably-healthy-cricket



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 62b59d52657713ea1b003a2d

Use following command to get test execution details:
$ kubectl testkube get execution 62b59d52657713ea1b003a2d


$ kubectl testkube get execution 62b59d52657713ea1b003a2d
ID:        62b59d52657713ea1b003a2d
Name:      sincerely-real-marten
Type:      kubepug/yaml
Duration:  00:00:05
Args:     --k8s-version=v1.18.0

{"DeprecatedAPIs":null,"DeletedAPIs":null}
Status Test execution completed with success 🥇
```

It is also possible to pass other arguments to the executor. For the options please consult the [KubePug documentation](https://github.com/kubepug/kubepug#how-to-use-it-as-a-standalone-program).
