# Artillery.io performance tests


## Test Environment

Let's assume that our SUT (Service Under Test) is an internal Kubernetes service which has
ClusterIP `Service` created and is exposed on port `8088`. The service name is `testkube-api-server`
and is exposing the `/health` endpoint that we want to test.

To call the SUT inside a cluster:

```sh
curl http://testkube-api-server:8088/health
```

Output:

```sh
200 OK
```

## Create a test manifest

The Artillery tests are defined in declarative manner, as YAML files.  
Let's save our test into `test.yaml` file with content below:   

```yaml
config:
  target: "http://testkube-api-server:8088/health"
  phases:
    - duration: 60
      arrivalRate: 5
      name: Warm up
    - duration: 120
      arrivalRate: 5
      rampTo: 50
      name: Ramp up load
    - duration: 600
      arrivalRate: 50
      name: Sustained load
```

Now we can create a new Testkube based on the saved Artillery test deifnition.

## **Create a New Testkube Test Script**

```sh
kubectl testkube create test --name artillery-api-test --file test.yaml --type artillery/test
```

Output:

```sh
Test created  🥇
```

## **Running a Test**

```sh
kubectl testkube run test artillery-api-test

Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d6398b046f8fbd3d955d4

or watch test execution until complete:
$ kubectl testkube watch execution 615d6398b046f8fbd3d955d4

```

Test runs can be named. If no name is passed, Testkube will autogenerate a name.

## **Getting Test Results**

Now we can watch/get test execution details:

```sh
kubectl testkube watch execution 615d6398b046f8fbd3d955d4
```

Output:

```sh
Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Watching for changes
Status: success, Duration: 598ms

Getting results
Name: openly-full-bream, Status: success, Duration: 598ms
newman

API-Health

→ Health
  GET http://testkube-api-server:8088/health [200 OK, 124B, 297ms]
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
│ total run duration: 523ms                                        │
├──────────────────────────────────────────────────────────────────┤
│ total data received: 8B (approx)                                 │
├──────────────────────────────────────────────────────────────────┤
│ average response time: 297ms [min: 297ms, max: 297ms, s.d.: 0µs] │
└──────────────────────────────────────────────────────────────────┘
Test execution completed in 598ms
```

## **Summary**

Testkube simplifies running tests inside a cluster and stores tests and tests results for later use.
