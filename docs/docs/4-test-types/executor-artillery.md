---
sidebar_position: 6
sidebar_label: Artillery.io
---
# Artillery.io Performance Tests


## **Test Environment**

Let's assume that our SUT (Service Under Test) is an internal Kubernetes service which has
ClusterIP `Service` created and is exposed on port `8088`. The service name is `testkube-api-server`
and is exposing the `/health` endpoint that we want to test.

To call the SUT inside a cluster:

```bash
curl http://testkube-api-server:8088/health
```

Output:

```bash
200 OK
```

## **Create a Test Manifest**

The Artillery tests are defined in declarative manner, as YAML files.  
The test should warm up our service a little bit first, then we can hit a little harder.

Let's save our test into `test.yaml` file with the content below:   

```yaml
config:
  target: "http://testkube-api-server:8088"
  phases:
    - duration: 6
      arrivalRate: 5
      name: Warm up
    - duration: 120
      arrivalRate: 5
      rampTo: 50
      name: Ramp up load
    - duration: 60
      arrivalRate: 50
      name: Sustained load
scenarios:
  - name: "Check health endpoint"
    flow:
      - get:
          url: "/health"
```

Our test is ready but how do we run it in a Kubernetes cluster? Testkube will help you with that! 

Let's create a new Testkube test based on the saved Artillery test definition.

## **Create a New Testkube Test**

```bash
kubectl testkube create test --name artillery-api-test --file test.yaml --type artillery/test
```

Output:

```bash
Test created  🥇
```

## **Running a Test**

```bash
kubectl testkube run test artillery-api-test

Type          : postman/collection
Name          : artillery-api-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d6398b046f8fbd3d955d4

or watch test execution until complete:
$ kubectl testkube watch execution 615d6398b046f8fbd3d955d4
```

You can also watch your test results in real-time with `-f` flag (like "follow"). 

Test runs can be named. If no name is passed, Testkube will autogenerate a name.

## **Getting Test Results**


Let's get back our finished test results. The test report and output will be stored in Testkube storage to revisit when necessary.

```bash
➜  testkube git:(jacek/docs/executors-docs-update) ✗ kubectl testkube get execution 628c957d2c8d8a7c1b1ead66                                                 
ID:        628c957d2c8d8a7c1b1ead66
Name:      tightly-adapting-hippo
Type:      artillery/test
Duration:  00:03:13

  Telemetry is on. Learn more: https://artillery.io/docs/resources/core/telemetry.html
Phase started: Warm up (index: 0, duration: 6s) 08:21:22(+0000)

Phase completed: Warm up (index: 0, duration: 6s) 08:21:28(+0000)

Phase started: Ramp up load (index: 1, duration: 120s) 08:21:28(+0000)

--------------------------------------
Metrics for period to: 08:21:30(+0000) (width: 6.167s)
--------------------------------------

http.codes.200: ................................................................ 41
http.request_rate: ............................................................. 9/sec
http.requests: ................................................................. 41
http.response_time:
  min: ......................................................................... 0
  max: ......................................................................... 5
  median: ...................................................................... 1
  p95: ......................................................................... 3
  p99: ......................................................................... 3
http.responses: ................................................................ 41
vusers.completed: .............................................................. 41
vusers.created: ................................................................ 41
vusers.created_by_name.Check health endpoint: .................................. 41
vusers.failed: ................................................................. 0
vusers.session_length:
  min: ......................................................................... 3.6
  max: ......................................................................... 73
  median: ...................................................................... 10.5
  p95: ......................................................................... 66
  p99: ......................................................................... 70.1

..... a lot of other .......


All VUs finished. Total time: 3 minutes, 9 seconds

--------------------------------
Summary report @ 08:24:30(+0000)
--------------------------------

http.codes.200: ................................................................ 6469
http.request_rate: ............................................................. 36/sec
http.requests: ................................................................. 6469
http.response_time:
  min: ......................................................................... 0
  max: ......................................................................... 17
  median: ...................................................................... 1
  p95: ......................................................................... 2
  p99: ......................................................................... 4
http.responses: ................................................................ 6469
vusers.completed: .............................................................. 6469
vusers.created: ................................................................ 6469
vusers.created_by_name.Check health endpoint: .................................. 6469
vusers.failed: ................................................................. 0
vusers.session_length:
  min: ......................................................................... 1.7
  max: ......................................................................... 73
  median: ...................................................................... 3
  p95: ......................................................................... 7.2
  p99: ......................................................................... 12.6
Log file: /tmp/test-report.json

Status Test execution completed with success 🥇

```

## **Summary**

With the Artillery executor you can now run your tests in Kubernetes with ease. Testkube simplifies running tests inside a cluster and stores tests and tests results for later use.
