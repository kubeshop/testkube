# Getting Results

We saw how simple it is to create and run Testkube tests executions. Obtaining test results is also simple.

## Getting Test Executions After Test is Executed

After each run, Testkube informs you that you can get results of a given test execution.

```sh
kubectl testkube run test api-incluster-test
```

```sh title="Expected output:"
Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d6398b046f8fbd3d955d4

Or watch test execution until complete:
$ kubectl testkube watch execution 615d6398b046f8fbd3d955d4

```

`testkube get execution 615d6398b046f8fbd3d955d4` - is for getting string output of test execution, where `615d6398b046f8fbd3d955d4` is the test execution ID.

## Change the Output Format of Execution Results

By default, Testkube returns string output of a particular executor. It can also return JSON or Go-Template based outputs.

### JSON Output

Sometimes you need to parse test results programmatically. To simplify this task, test execution results can be in JSON format.

```sh
testkube get execution 615d7e1ab046f8fbd3d955d6 -o json
```

```json title="Expected output:"
{
  "id": "615d7e1ab046f8fbd3d955d6",
  "testName": "api-incluster-test",
  "testType": "postman/collection",
  "name": "monthly-sure-finch",
  "executionResult": {
    "status": "passed",
    "startTime": "2021-10-06T10:44:46.338Z",
    "endTime": "2021-10-06T10:44:46.933Z",
    "output": "newman\n\nAPI-Health\n\n→ Health\n  GET http://testkube-api-server:8088/health [200 OK, 124B, 282ms]\n  ✓  Status code is 200\n\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-tests │                  2 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-tests │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │                 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 519ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 8B (approx)                                 │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 282ms [min: 282ms, max: 282ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n",
    "outputType": "text/plain",
    "steps": [
      {
        "name": "Health",
        "duration": "282ms",
        "status": "passed",
        "assertionResults": [
          { "name": "Status code is 200", "status": "passed" }
        ]
      }
    ]
  }
}
```

It is quite easy to parse data from test executions with tools like `jq` or in other programmatic ways.

### Need Non-standard Output? Go-Template for the Rescue

If you need non-standard test execution output, you can easily use output `-o go` with the passed `--go-template` template content.

```sh
testkube get execution 615d7e1ab046f8fbd3d955d6 -ogo --go-template='{{.Name}} {{.Id}} {{.ExecutionResult.Status}}'
```

```sh title="Expected output:"
monthly-sure-finch 615d7e1ab046f8fbd3d955d6 success
```

## Getting a List of Test Executions

<!--- Please watch this video on getting tests results in different formats:

<iframe width="560" height="315" src="https://www.youtube.com/embed/ukHvS5x7TvM" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe> --->

### Getting a List of Recent Executions

We can get a list of current executions with use of the `executions` subcommand:

```sh
testkube get executions
```

```sh title="Expected output:"
        TEST          |        TYPE        | NAME |            ID            | STATUS
+---------------------+--------------------+------+--------------------------+---------+
  api-incluster-test  | postman/collection |      | 615d7e1ab046f8fbd3d955d6 | success
  api-incluster-test  | postman/collection |      | 615d6398b046f8fbd3d955d4 | success
  kubeshop-cypress    | cypress/project    |      | 615d5372b046f8fbd3d955d2 | success
  kubeshop-cypress    | cypress/project    |      | 615d5265b046f8fbd3d955d0 | error
  cypress-example     | cypress/project    |      | 615d4fe6b046f8fbd3d955ce | error
  cypress-example     | cypress/project    |      | 615d4556b046f8fbd3d955cc | error
```

Now we can use an ID to check the results:

```sh
testkube get execution 615d5265b046f8fbd3d955d0
```

### Getting a List of Executions in Different Formats

Terminal mode table data is not always best when processing results in code or shell tests. To simplify this, we have implemented JSON or Go-Template based results when getting results lists.

#### JSON

Getting JSON results is quite easy, just pass the `-o json` flag to the command:

```sh
testkube get executions -o json
```

```json title="Expected output:"
{
  "totals": {
    "results": 17,
    "passed": 7,
    "failed": 10,
    "queued": 0,
    "pending": 0
  },
  "results": [
    {
      "id": "615d7e1ab046f8fbd3d955d6",
      "name": "",
      "testName": "api-incluster-test",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-10-06T10:44:46.338Z",
      "endTime": "2021-10-06T10:44:46.933Z"
    },
    {
      "id": "615d6398b046f8fbd3d955d4",
      "name": "",
      "testName": "api-incluster-test",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-10-06T08:51:39.834Z",
      "endTime": "2021-10-06T08:51:40.432Z"
    },
    {
      "id": "615d5372b046f8fbd3d955d2",
      "name": "",
      "testName": "kubeshop-cypress",
      "testType": "cypress/project",
      "status": "passed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-10-06T07:44:30.025Z"
    },
    {
      "id": "615d5265b046f8fbd3d955d0",
      "name": "",
      "testName": "kubeshop-cypress",
      "testType": "cypress/project",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-10-06T07:40:09.261Z"
    },
    {
      "id": "615d4fe6b046f8fbd3d955ce",
      "name": "",
      "testName": "cypress-example",
      "testType": "cypress/project",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-10-06T07:28:54.579Z"
    },
    {
      "id": "615d4556b046f8fbd3d955cc",
      "name": "",
      "testName": "cypress-example",
      "testType": "cypress/project",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-10-06T06:43:44.1Z"
    },
    {
      "id": "615d43d3b046f8fbd3d955ca",
      "name": "",
      "testName": "cypress-example",
      "testType": "cypress/project",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-10-06T06:37:52.601Z"
    },
    {
      "id": "6155cd7db046f8fbd3d955c8",
      "name": "",
      "testName": "postman-test-7f6qrm",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-09-30T14:45:20.819Z",
      "endTime": "2021-09-30T14:45:21.419Z"
    },
    {
      "id": "6155cd67b046f8fbd3d955c6",
      "name": "",
      "testName": "sanity",
      "testType": "postman/collection",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-30T14:45:00.135Z"
    },
    {
      "id": "615322f3f47de75f31ae7a06",
      "name": "",
      "testName": "long-1",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-09-28T14:13:11.293Z",
      "endTime": "2021-09-28T14:13:45.271Z"
    },
    {
      "id": "61532298f47de75f31ae7a04",
      "name": "",
      "testName": "long-1",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-09-28T14:11:39.179Z",
      "endTime": "2021-09-28T14:12:15.202Z"
    },
    {
      "id": "6151b4b342189df67944968e",
      "name": "",
      "testName": "postman-test-7f6qrm",
      "testType": "postman/collection",
      "status": "passed",
      "startTime": "2021-09-27T12:10:31.581Z",
      "endTime": "2021-09-27T12:10:32.105Z"
    },
    {
      "id": "6151b49d42189df67944968c",
      "name": "",
      "testName": "curl-test",
      "testType": "curl/test",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-27T12:10:06.954Z"
    },
    {
      "id": "6151b41742189df67944968a",
      "name": "",
      "testName": "curl-test",
      "testType": "curl/test",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-27T12:07:52.893Z"
    },
    {
      "id": "6151b41342189df679449688",
      "name": "",
      "testName": "curl-test",
      "testType": "curl/test",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-27T12:07:48.868Z"
    },
    {
      "id": "6151b40f42189df679449686",
      "name": "",
      "testName": "curl-test",
      "testType": "curl/test",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-27T12:07:44.89Z"
    },
    {
      "id": "6151b40b42189df679449684",
      "name": "",
      "testName": "curl-test",
      "testType": "curl/test",
      "status": "failed",
      "startTime": "0001-01-01T00:00:00Z",
      "endTime": "2021-09-27T12:07:41.168Z"
    }
  ]
}
```

#### Go-Template

To get a list of test execution IDs with their corresponding statuses with go-template:

```sh
testkube get executions -ogo --go-template '{{.Id}}:{{.Status}} '
```

```sh title="Expected output"
 615d7e1ab046f8fbd3d955d6:success 615d6398b046f8fbd3d955d4:success 615d5372b046f8fbd3d955d2:success 615d5265b046f8fbd3d955d0:error 615d4fe6b046f8fbd3d955ce:error 615d4556b046f8fbd3d955cc:error 615d43d3b046f8fbd3d955ca:error 6155cd7db046f8fbd3d955c8:success 6155cd67b046f8fbd3d955c6:error 615322f3f47de75f31ae7a06:success 61532298f47de75f31ae7a04:success 6151b4b342189df67944968e:success 6151b49d42189df67944968c:error 6151b41742189df67944968a:error 6151b41342189df679449688:error 6151b40f42189df679449686:error 6151b40b42189df679449684:error
```

### Getting a List of Executions of a Given Test

To find the execution of a particular test, pass the test name as a parameter:

```sh
testkube get executions api-incluster-test
```

```sh title="Expected output:"
        TEST         |        TYPE        | NAME |            ID            | STATUS
+--------------------+--------------------+------+--------------------------+---------+
  api-incluster-test | postman/collection |      | 615d6398b046f8fbd3d955d4 | success
  api-incluster-test | postman/collection |      | 615d7e1ab046f8fbd3d955d6 | success
```

### Getting the Test Status of a Given Test from Test CRD

To get the Test CRD status of a particular test, pass the test name as a parameter:

```sh
testkube get tests container-test --crd-only
```

```yaml title="Expected output:"
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: container-test
  namespace: testkube
spec:
  type: container/test
  content:
    type: string
    data: ""
  executionRequest:
    artifactRequest:
      storageClassName: standard
      volumeMountPath: /share
      dirs:
      - test/files
status:
  latestExecution:
    id: 63b755cab2a16c73e8cfa1c4
    number: 1
    startTime: 2023-01-05T22:57:14Z
    endTime: 2023-01-05T22:57:28Z
    status: passed
```
