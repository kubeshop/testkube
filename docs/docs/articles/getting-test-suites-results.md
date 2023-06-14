# Getting Results

To get recent results, call the `testsuites executions` subcommand:

```sh
testkube get tse
```

```sh title="Expected output:"

  ID                       | TEST SUITE NAME                | EXECUTION NAME                        | STATUS  | STEPS | LABELS        
---------------------------+--------------------------------+---------------------------------------+---------+-------+---------------
  63d401e5fed6933f342ccc67 | executor-maven-smoke-tests     | ts-executor-maven-smoke-tests-680     | failed  |     3 | app=testkube  
  63d401a9fed6933f342ccc61 | executor-artillery-smoke-tests | ts-executor-artillery-smoke-tests-682 | passed  |     2 | app=testkube  
  63d3fed9fed6933f342ccc5b | executor-jmeter-smoke-tests    | ts-executor-jmeter-smoke-tests-500    | passed  |     2 | app=testkube  
  63d3fd35fed6933f342ccc51 | executor-postman-smoke-tests   | ts-executor-postman-smoke-tests-671   | passed  |     4 | app=testkube  
  63d3fb91fed6933f342ccc4b | executor-container-smoke-tests | ts-executor-container-smoke-tests-683 | failed  |     2 | app=testkube

```

## **Getting a Single Test Suite Execution**

With the test suite execution ID, you can get single test suite results:

```sh
testkube get tse 61e1136165e59a3183465125
```

```sh title="Expected output:"
Id:       63d3cd05c6768fc8b574e2e8
Name:     ts-testsuite-parallel-19
Status:   passed
Duration: 22.138s

Labels:   
  STATUSES               | STEP                           | IDS                            | ERRORS      
-------------------------+--------------------------------+--------------------------------+-------------
  passed, passed, passed | run:testkube/cli-test,         | 63d3cd05c6768fc8b574e2e9,      | "", "", ""  
                         | run:testkube/demo-test, delay  | 63d3cd05c6768fc8b574e2ea, ""   |             
                         | 1000ms                         |                                |             
  passed                 | delay 5000ms                   | ""                             | ""          


Use the following command to get test suite execution details:
$ kubectl testkube get tse 61e1136165e59a3183465125
```

Test Suite steps that are running workflows based on `Test` Custom Resources have a Test Execution ID. You can get the details of each in a separate command:

```sh 
kubectl testkube get execution 63d3cd05c6768fc8b574e2e9

ID:         63d3cd05c6768fc8b574e2e9
Name:       testsuite-parallel-cli-test-46
Number:            46
Test name:         cli-test
Type:              cli/test
Status:            passed
Start time:        2023-01-27 13:09:25.54 +0000 UTC
End time:          2023-01-27 13:09:42.432 +0000 UTC
Duration:          00:00:16


TODO

→ Create TODO
  POST http://34.74.127.60:8080/todos [201 Created, 296B, 100ms]
  ✓  Status code is 201 CREATED
  ┌
  │ 'creating', 'http://34.74.127.60:8080/todos/50'
  └
  ✓  Check if todo item created successfully
  GET http://34.74.127.60:8080/todos/50 [200 OK, 291B, 8ms]

→ Complete TODO item
  ┌
  │ 'completing', 'http://34.74.127.60:8080/todos/50'
  └
  PATCH http://34.74.127.60:8080/todos/50 [200 OK, 290B, 8ms]

→ Delete TODO item
  ┌
  │ 'deleting', 'http://34.74.127.60:8080/todos/50'
  └
  DELETE http://34.74.127.60:8080/todos/50 [204 No Content, 113B, 7ms]
  ✓  Status code is 204 no content

┌─────────────────────────┬───────────────────┬──────────────────┐
│                         │          executed │           failed │
├─────────────────────────┼───────────────────┼──────────────────┤
│              iterations │                 1 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│                requests │                 4 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│            test-scripts │                 5 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│      prerequest-scripts │                 6 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│              assertions │                 3 │                0 │
├─────────────────────────┴───────────────────┴──────────────────┤
│ total run duration: 283ms                                      │
├────────────────────────────────────────────────────────────────┤
│ total data received: 353B (approx)                             │
├────────────────────────────────────────────────────────────────┤
│ average response time: 30ms [min: 7ms, max: 100ms, s.d.: 39ms] │
└────────────────────────────────────────────────────────────────┘

```

### Getting a Test Suite Status of a Given Test Suite from Test Suite CRD

To get the Test Suite CRD status of a particular test suite, pass the test suite name as a parameter:

```sh
kubectl testkube get testsuites test-suite-example --crd-only
```

Output:

```yaml title="Expected output:"
apiVersion: tests.testkube.io/v3
kind: TestSuite
metadata:
  name: test-suite-example
  namespace: testkube
spec:
  steps:
  - stopOnFailure: false
    execute:
    - test: testkube-dashboard
    - delay: 1s
    - test: testkube-homepage

status:
  latestExecution:
    id: 63b7551cb2a16c73e8cfa1bf
    startTime: 2023-01-05T22:54:20Z
    endTime: 2023-01-05T22:54:29Z
    status: failed
```
