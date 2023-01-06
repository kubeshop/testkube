---
sidebar_position: 3
sidebar_label: Getting results
---

# Getting a List of Recent Test Executions

To get recent results, call the `tests executions` subcommand:

```bash

kubectl testkube get tse

             ID            |  TEST NAME   |           EXECUTION NAME            | STATUS  | STEPS  
+--------------------------+--------------+-------------------------------------+---------+-------+
  61e1142465e59a318346512b | test-example | test-example.equally-enabled-heron  | success |     3  
  61e1136165e59a3183465125 | test-example | test-example.fairly-humble-tick     | success |     3  
  61dff61867326ad521b2a0d6 | test-example | test-example.verbally-merry-hagfish | success |     3  
  61dfe0de69b7bfcb9058dad0 | test-example | test-example.overly-exciting-krill  | success |     3  

```


## **Getting a Single Test Execution**

With the test execution ID, you can get single test results:

```bash 
kubectl testkube get tse 61e1136165e59a3183465125 

Name: test-example.fairly-humble-tick
Status: success

             STEP            | STATUS  |            ID            | ERROR  
+----------------------------+---------+--------------------------+-------+
  run test: testkube/test1 | success | 61e1136165e59a3183465127 |        
  delay 2000ms               | success |                          |        
  run test: testkube/test1 | success | 61e1136765e59a3183465129 |        



Use the following command to get test execution details:
$ kubectl testkube get tse 61e1136165e59a3183465125
```

Test Suite steps that are running workflows based on `Test` Custom Resources have a Test Execution ID. You can get the details of each in a separate command: 

```bash 
kubectl testkube get execution 61e1136165e59a3183465127Name: test-example-test1, Status: success, Duration: 4.677s

newman

TODO

→ Create TODO
  POST http://34.74.127.60:8080/todos [201 Created, 296B, 100ms]
  ✓  Status code is 201 CREATED
  ┌
  │ 'creating', 'http://34.74.127.60:8080/todos/50'
  └
  ✓  Check if todo item craeted successfully
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

### **Getting a Test Suite status of a Given Test Suite from Test Suite CRD**

To get the Test Suite CRD status of a particular test suite, pass the test suite name as a parameter:

```bash
kubectl testkube get testsuites test-suite-example --crd-only
```

Output:

```bash
apiVersion: tests.testkube.io/v2
kind: TestSuite
metadata:
  name: test-suite-example
  namespace: testkube
spec:
  steps:
    execute:
      stopOnFailure: false
      namespace: testkube
      name: test-case
status:
  latestExecution:
    id: 63b7551cb2a16c73e8cfa1bf
    startTime: 2023-01-05T22:54:20Z
    endTime: 2023-01-05T22:54:29Z
    status: failed
```
