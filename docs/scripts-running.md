# Running Testkube Test Scripts

Test scripts are stored in Kubernetes cluster as Custom resources. We can run them as many times as we want and get results with use of kubectl testkube plugin or with an API.

## **Running Tests**

Running scripts looks the same for any type of script.
Let's assume we've previously created a script with the name `api-incluster-test`.

### **Standard Run Command**

This is the simplest run command:

```sh
kubectl testkube run test api-incluster-test
```

Output:

```sh
Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Script queued for execution
Use the following command to get script execution details:
$ kubectl testkube get execution 615d6398b046f8fbd3d955d4

Or watch script execution until complete:
$ kubectl testkube watch execution 615d6398b046f8fbd3d955d4

```

Testkube will inform us about possible commands to get scripts:

- `kubectl testkube get execution 615d6398b046f8fbd3d955d4` to get execution details.
- `kubectl testkube watch execution 615d6398b046f8fbd3d955d4` to watch current pending executions. Watch will also get the details when the script is completed and will lock the terminal until long running scripts complete.

## **Run with Watch for Changes**

If we want to wait until script execution completes, we can pass `-f` flag (follow) to the script run command.

```sh
kubectl testkube run test api-incluster-test -f
```

Output:

```sh
Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d7e1ab046f8fbd3d955d6
Execution name: monthly-sure-finch

Script queued for execution

Use the following command to get script execution details:
$ kubectl testkube get execution 615d7e1ab046f8fbd3d955d6

Or watch script execution until complete:
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
Use following command to get script execution details:
$ kubectl testkube get execution 615d7e1ab046f8fbd3d955d6

Script execution completed in 595ms
```

This command will wait until script execution completes with an error or success.

### **Passing Parameters**

For some 'real world' tests, you need to pass configuration variables to run them on different environments or with different test configuration.

Let's assume that our example Cypress test needs the `testparam` parameter with the value `testvalue`.

Use the `-p` parameter to pass this configuration. If you need to pass more than one parameter, simply pass multiple `-p` flags:

```sh
kubectl testkube run test kubeshop-cypress -p testparam=testvalue -f
```

Output:

```sh
Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5372b046f8fbd3d955d2
Execution name: nominally-able-glider

Script queued for execution

Use the following command to get script execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

Or watch script execution until complete:
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

Use the following command to get script execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

Script execution completed in 1m45.405939s
```

## **Summary**

Running scripts in Kubernetes cluster is really easy with the use of the Testkube kubectl plugin!
