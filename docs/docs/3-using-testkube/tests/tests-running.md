---
sidebar_position: 3
sidebar_label: Running
---

# Running Testkube Tests

Tests are stored in Kubernetes clusters as Custom Resources. Testkube tests are reusable and can get results with the use of kubectl testkube plugin or with an API.

## **Running**

Running tests looks the same for any type of test.
In this example, we have previously created a test with the name `api-incluster-test`.

### **Standard Run Command**

This is an example of the simplest run command:

```bash
kubectl testkube run test api-incluster-test
```

Output:

```bash
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

Testkube will inform us about possible commands to get test results:

- `kubectl testkube get execution 615d6398b046f8fbd3d955d4` to get execution details.
- `kubectl testkube watch execution 615d6398b046f8fbd3d955d4` to watch the current pending execution. Watch will also get details when a test is completed and is good for long running tests to lock your terminal until test execution completes.

## **Run with Watch for Changes**

If we want to wait until a test execution completes we can pass the `-f` flag (follow) to the test run command:

```bash
kubectl testkube run test api-incluster-test -f
```

Output:

```bash
Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d7e1ab046f8fbd3d955d6
Execution name: monthly-sure-finch

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d7e1ab046f8fbd3d955d6

Or watch test execution until complete:
$ kubectl testkube watch execution 615d7e1ab046f8fbd3d955d6


Watching for changes
Status: pending, Duration: 222.387ms
Status: pending, Duration: 1.210689s
Status: pending, Duration: 2.201346s
Status: pending, Duration: 3.198539s
Status: success, Duration: 595ms

Getting results
Name: monthly-sure-finch, Status: success, Duration: 595ms
newman

API-Health

→ Health
  GET http://testkube-api-server:8088/health [200 OK, 124B, 282ms]
  ✓  Status code is 200

┌─────────────────────────┬────────────────────┬───────────────────┐
│                         │           executed │            failed │
├─────────────────────────┼────────────────────┼───────────────────┤
│              iterations │                  1 │                 0 │
├─────────────────────────┼────────────────────┼───────────────────┤
│                requests │                  1 │                 0 │
├─────────────────────────┼────────────────────┼───────────────────┤
│            test-scripts │                  2 │                 0 │
├─────────────────────────┼────────────────────┼───────────────────┤
│      prerequest-scripts │                  1 │                 0 │
├─────────────────────────┼────────────────────┼───────────────────┤
│              assertions │                  1 │                 0 │
├─────────────────────────┴────────────────────┴───────────────────┤
│ total run duration: 519ms                                        │
├──────────────────────────────────────────────────────────────────┤
│ total data received: 8B (approx)                                 │
├──────────────────────────────────────────────────────────────────┤
│ average response time: 282ms [min: 282ms, max: 282ms, s.d.: 0µs] │
└──────────────────────────────────────────────────────────────────┘
Use the following command to get test execution details:
$ kubectl testkube get execution 615d7e1ab046f8fbd3d955d6

Test execution completed in 595ms
```

This command will wait until the test execution completes.

### **Passing Parameters**

For some 'real world' tests, configuration variables are passed in order to run them on different environments or with different test configurations.

Let's assume that our example Cypress test needs the `testparam` parameter with the value `testvalue`.

This is done by using the `-p` parameter. If you need to pass more parameters, simply pass multiple `-p` flags.

It's possible to pass parameters securely to the executed test. It's necessary to use `--secret` flag,
which contains a key value pair - a name of the kubernetes secret and a secret key.
Pass it multiple times if needed.

```bash
kubectl testkube run test kubeshop-cypress -p testparam=testvalue -f --secret secret-name=secret-key
```

Output:

```bash
Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5372b046f8fbd3d955d2
Execution name: nominally-able-glider

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

or watch test execution until complete:
$ kubectl testkube watch execution 615d5372b046f8fbd3d955d2


Watching for changes
Status: queued, Duration: 0s
Status: pending, Duration: 383.064ms
....
Status: pending, Duration: 1m45.405939s
Status: success, Duration: 1m45.405939s

Getting results
Name: nominally-able-glider, Status: success, Duration: 2562047h47m16.854775807s

====================================================================================================

  (Run Starting)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Cypress:    8.5.0                                                                              │
  │ Browser:    Electron 91 (headless)                                                             │
  │ Specs:      1 found (simple-test.js)                                                           │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


────────────────────────────────────────────────────────────────────────────────────────────────────

  Running:  simple-test.js                                                                  (1 of 1)

  (Results)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Tests:        1                                                                                │
  │ Passing:      1                                                                                │
  │ Failing:      0                                                                                │
  │ Pending:      0                                                                                │
  │ Skipped:      0                                                                                │
  │ Screenshots:  0                                                                                │
  │ Video:        true                                                                             │
  │ Duration:     19 seconds                                                                       │
  │ Spec Ran:     simple-test.js                                                                   │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


  (Video)

  -  Started processing:  Compressing to 32 CRF
    Compression progress:  39%
    Compression progress:  81%
  -  Finished processing: /tmp/testkube-scripts531364188/repo/examples/cypress/videos/   (30 seconds)
                          simple-test.js.mp4

    Compression progress:  100%

====================================================================================================

  (Run Finished)


       Spec                                              Tests  Passing  Failing  Pending  Skipped
  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ ✔  simple-test.js                           00:19        1        1        -        -        - │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘
    ✔  All specs passed!                        00:19        1        1        -        -        -

Use the following command to get test execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

Test execution completed in 1m45.405939s
```

### **Mapping local files**

Local files can be set on the execution of a Testkube Test. Pass the file in the format `source_path:destination_path` using the flag `--copy-files`.

```bash
kubectl testkube run test maven-example-file-test --copy-files "/Users/local_user/local_maven_settings.xml:/tmp/settings.xml" --args "--settings" --args "/tmp/settings.xml" -v "TESTKUBE_MAVEN=true"
```

## **Summary**

As we can see, running tests in Kubernetes cluster is really easy with use of the Testkube kubectl plugin!
