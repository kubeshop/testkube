import Admonition from "@theme/Admonition";

# Tracetest

Testkube supports running trace-based tests with Tracetest.

<iframe width="100%" height="250" src="https://www.youtube.com/embed/nAp3zYgykok" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

export const ExecutorInfo = () => {
  return (
    <div>
      <Admonition type="info" icon="🎓" title="What is Tracetest?">
        <ul>
          <li><a href="https://tracetest.io/">Tracetest</a> is a OpenTelemetry based tool that helps you develop and test your distributed applications. It assists you in the development process by enabling you to trigger your code and see the trace as you add OTel instrumentation.</li>
        </ul>
        <b>What can I do with Tracetest?</b>
        <ul>
          <li>Tracetest uses your existing OpenTelemetry traces to power trace-based testing with assertions against your trace data at every point of the request transaction.</li>
        </ul>
      </Admonition>
    </div>
  );
}

<ExecutorInfo />

By using the Testkube Tracetest Executor you can now add Tracetest to the native CI/CD pipeline in your Kubernetes cluster. This allows you to add all the additional Testkube functionalities, like running scheduled test runs and using Test Triggers. All while following the trace-based testing principle and enabling full in-depth assertions against trace data, not just the response.

## **Running a Tracetest Test**

In order to run Tracetest tests using Testkube, you will need to have the following in your Kubernetes cluster:
1. Testkube
2. Tracetest
3. An OpenTelemetry Instrumented Service

To install Tracetest, refer to
[Tracetest's documentation](https://docs.tracetest.io/getting-started/installation/). 

To create the test in Testkube, you'll need to export your Tracetest `Test Definition`. An example of an exported Tracetest definition looks like the following:

```yaml
type: Test
spec:
  id: RUkKQ_aVR
  name: Pokeshop - List
  description: Get a Pokemon
  trigger:
    type: http
    httpRequest:
      url: http://demo-pokemon-api.demo/pokemon?take=20&skip=0
      method: GET
      headers:
      - key: Content-Type
        value: application/json
  specs:
  - name: Database queries less than 500 ms
    selector: span[tracetest.span.type="database"]
    assertions:
    - attr:tracetest.span.duration  <  500ms
```

### **Add Test to Testkube**

To add your Tracetest tests to Testkube, use the `create test` command. You'll need to reference the `.yaml` file for your test definitions using the `--file` argument and the Tracetest Server endpoint using the `--variable` argument with `TRACETEST_ENDPOINT`.

Your Tracetest Endpoint should be reachable by Testkube in your cluster. Use your Tracetest service's `CLUSTER-IP:PORT`, for example: `10.96.93.106:11633`.

```bash
kubectl testkube create test --file ./test.yaml --type "tracetest/test" --name tracetest-test --variable TRACETEST_ENDPOINT=http://CLUSTER-IP:PORT
```

```sh title="Expected output:"
Test created testkube / tracetest-test 🥇
```

### **Running Tracetest Tests in Testkube**

To execute the test, run the following command:

```bash
kubectl testkube run test --watch tracetest-test
```

Alternatively, you can also run the test from the Testkube Dashboard.

### **Getting Test Results**

If the test passes, the Testkube CLI will look like this:

```sh title="Expected output:"
Type:              tracetest/test
Name:              tracetest-test
Execution ID:      6418873d9922b3e1003dd5b8
Execution name:    tracetest-test-4
Execution number:  4
Status:            running
Start time:        2023-03-20 16:18:05.60245717 +0000 UTC
End time:          0001-01-01 00:00:00 +0000 UTC
Duration:

  Variables:    1
  - TRACETEST_ENDPOINT = http://10.96.93.106:11633


Getting logs from test job 6418873d9922b3e1003dd5b8
Execution completed
🔬 Executing in directory :
 $ tracetest test run --server-url http://10.96.93.106:11633 --definition /tmp/test-content1901459587 --wait-for-result --output pretty
✔ Pokeshop - List (http://10.96.93.106:11633/test/RUkKQ_aVR/run/3/test)
    ✔ Database queries less than 500 ms

✅ Execution succeeded
Execution completed ✔ Pokeshop - List (http://10.96.93.106:11633/test/RUkKQ_aVR/run/3/test)
    ✔ Database queries less than 500 ms
```

Alternatively, if it fails:

```sh title="Expected output:"
Type:              tracetest/test
Name:              tracetest-test
Execution ID:      641885f39922b3e1003dd5b6
Execution name:    tracetest-test-3
Execution number:  3
Status:            running
Start time:        2023-03-20 16:12:35.268197087 +0000 UTC
End time:          0001-01-01 00:00:00 +0000 UTC
Duration:

  Variables:    1
  - TRACETEST_ENDPOINT = http://10.96.93.106:11633

Getting logs from test job 641885f39922b3e1003dd5b6
Execution completed
🔬 Executing in directory :
 $ tracetest test run --server-url http://10.96.93.106:11633 --definition /tmp/test-content737616681 --wait-for-result --output pretty
✘ Pokeshop - List (http://10.96.93.106:11633/test/RUkKQ_aVR/run/2/test)
    ✘ Database queries less than 500 ms
        ✘ #2b213392d0e3ff21
            ✘ attr:tracetest.span.duration  <  500ms (502ms) (http://10.96.93.106:11633/test/RUkKQ_aVR/run/2/test?selectedAssertion=0&selectedSpan=2b213392d0e3ff21)
        ✔ #7e6657f6a43fceeb
            ✔ attr:tracetest.span.duration  <  500ms (72ms)
        ✔ #6ee2fb69690eed47
            ✔ attr:tracetest.span.duration  <  500ms (13ms)
        ✘ #a82c304a3558763b
            ✘ attr:tracetest.span.duration  <  500ms (679ms) (http://10.96.93.106:11633/test/RUkKQ_aVR/run/2/test?selectedAssertion=0&selectedSpan=a82c304a3558763b)
        ✔ #6ae21f2251101fd6
            ✔ attr:tracetest.span.duration  <  500ms (393ms)
        ✔ #2a9b9422af8ba1a8
            ✔ attr:tracetest.span.duration  <  500ms (61ms)
        ✔ #010a8a0d53687276
            ✔ attr:tracetest.span.duration  <  500ms (36ms)
        ✘ #895d66286b6325ae
            ✘ attr:tracetest.span.duration  <  500ms (686ms) (http://10.96.93.106:11633/test/RUkKQ_aVR/run/2/test?selectedAssertion=0&selectedSpan=895d66286b6325ae)
```

## **References**

- [Tracetest Documentation](https://docs.tracetest.io/)
- [Testkube Tracetest Executor - GitHub Repo](https://github.com/kubeshop/testkube-executor-tracetest)
- [Blog: Running Scheduled Trace-based Tests with Tracetest and Testkube](https://docs.tracetest.io/examples-tutorials/recipes/running-tracetest-with-testkube/)
