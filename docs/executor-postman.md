# Postman Collections

Watch this simple TestKube intro video for Postman collections with TestKube:

<iframe width="560" height="315" src="https://www.youtube.com/embed/rWqlbVvd8Dc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen>
</iframe>

TestKube is able to run Postman collections inside your Kubernetes cluster so it can be used to test internal or external services.

## Test Environment

Let's assume that our SUT (Service Under Test) is an internal Kubernetes service which has a 
NodePort `Service` created and is exposed on port `8088`. The service name is `testkube-api-server`
and is exposing the `/health` endpoint that we want to test.

To call the SUT inside a cluster:

```sh
curl http://testkube-api-server:8088/health
```

Output:

```sh
200 OK 
```

## Create a New Postman Test

Create a postman collection and export it as JSON:

![postman create collection](img/postman_create_collection.png)

Right click and export the given Collection to a file,
In this example, it is saved into `~/Downloads/API-Health.postman_collection.json`

Now we can create a new TestKube based on the saved Postman Collection.

## Create a New TestKube Test Script

```sh
kubectl testkube scripts create --name api-incluster-test --file ~/Downloads/API-Health.postman_collection.json --type postman/collection 
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Script created  🥇
```

Script created!

## Running a Test

```sh
kubectl testkube scripts run api-incluster-test

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Script queued for execution.
Use the following command to get script execution details:
$ kubectl testkube scripts execution 615d6398b046f8fbd3d955d4

Or watch script execution until complete:
$ kubectl testkube scripts watch 615d6398b046f8fbd3d955d4

```

You can name your test runs. If no name is passed, TestKube will autogenerate a name.

## Getting Test Results

Now we can watch/get script execution details:

```sh
kubectl testkube scripts watch 615d6398b046f8fbd3d955d4
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
Script execution completed in 598ms
```

## Summary

TestKube simplifies running tests inside a cluster and stores our tests and tests results for later use.
