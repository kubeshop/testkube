---
sidebar_position: 2
sidebar_label: Postman
---
# Postman Collections

<!-- Watch this simple Testkube intro video for Postman collections with Testkube:

<iframe width="560" height="315" src="https://www.youtube.com/embed/rWqlbVvd8Dc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen>
</iframe> -->

Testkube is able to run Postman collections inside your Kubernetes cluster so it can be used to test internal or external services.


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

## **Create a New Postman Test**

Create a postman collection and export it as JSON:

![postman create collection](../img/postman_create_collection.png)

Right click and export the given Collection to a file,
In this example, it is saved into `~/Downloads/API-Health.postman_collection.json`

Now we can create a new Testkube based on the saved Postman Collection.

## **Create a New Testkube Test Script**

```bash
kubectl testkube create test --name api-incluster-test --file ~/Downloads/API-Health.postman_collection.json --type postman/collection
```

Output:

```bash
Test created  🥇
```

Test created!

## **Running a Test**

```bash
kubectl testkube run test api-incluster-test

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

```bash
kubectl testkube watch execution 615d6398b046f8fbd3d955d4
```

Output:

```bash
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
