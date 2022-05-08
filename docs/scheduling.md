# Testkube Test and Test Suite Scheduling

In order to run Tests and Test Suites on a regular basis, we support a scheduling mechanism for these objects.
CRDs both for test and test suite contain a **schedule** field used to define rules for launching them in time.
Testkube's schedule data format is the same that is used to define Kubernetes Cron jobs (check Wikipedia Cron format for details https://en.wikipedia.org/wiki/Cron).

## Scheduling Architecture

Testkube uses the scheduling engine from Kubernetes Cron jobs.
In fact, for each scheduled Test or Test Suite, a special cron job is created from this template:
https://github.com/kubeshop/helm-charts/blob/main/charts/testkube-api/cronjob_template.yml.
Technically, it is a callback to Testkube API server method launching either Test or Test Suite execution.
This works similar to scheduled test and test suite executions done by external scheduling platforms. 

## Create a Test with a Schedule

Let's create a test with a required schedule using Testkube CLI command:

```sh
kubectl testkube create test --file test/postman/TODO.postman_collection.json --name scheduled-test --schedule="*/1 * * * *"
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████
                                           /tɛst kjub/ by Kubeshop


Detected test type postman/collection
Test created  / scheduled-test 🥇
```

We successfuly created a scheduled test and can check a list of the available tests:

```sh
kubectl testkube get tests
```

Output:

```sh
  NAME              | TYPE               | CREATED                       | LABELS | SCHEDULE    | STATUS | EXECUTION ID              
+-------------------+--------------------+-------------------------------+--------+-------------+--------+--------------------------+
  scheduled-test    | postman/collection | 2022-04-13 12:37:40 +0000 UTC |        | */1 * * * * |        |                           
```

The scheduled test was created but it was not yet executed. 

## Run a Scheduled Test

In order to start execution of the test on a defined schedule, run it using the Testkube CLI command:

```sh
kubectl testkube run test scheduled-test
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : postman/collection
Name          : scheduled-test



Test queued for execution

Use the following command to get test execution details:
$ kubectl testkube get execution 
```

Test was successfully scheduled for execution and put into the queue.
Let's check a Cron job connected to this test.

```sh
kubectl get cronjobs -A
```

Output:

```sh
NAMESPACE   NAME                   SCHEDULE      SUSPEND   ACTIVE   LAST SCHEDULE   AGE
testkube    scheduled-test-tests   */1 * * * *   False     1        42s           3m22s
```

The Cron job for this test was successfully created and the test was executed.
To check Cron job details:

```sh
kubectl describe cronjob scheduled-test-tests -n testkube
```

Output:

```sh
Name:                          scheduled-test-tests
Namespace:                     testkube
Labels:                        testkube=tests
Annotations:                   <none>
Schedule:                      */1 * * * *
Concurrency Policy:            Forbid
Suspend:                       False
Successful Job History Limit:  3
Failed Job History Limit:      1
Starting Deadline Seconds:     <unset>
Selector:                      <unset>
Parallelism:                   <unset>
Completions:                   <unset>
Pod Template:
  Labels:  <none>
  Containers:
   curlimage:
    Image:      curlimages/curl
    Port:       <none>
    Host Port:  <none>
    Command:
      sh
      -c
    Args:
      curl -X POST -H "Content-Type: application/json" -d '{}' "http://testkube-api-server:8088/v1/tests/scheduled-test/executions?callback=true"
    Environment:     <none>
    Mounts:          <none>
  Volumes:           <none>
Last Schedule Time:  Wed, 13 Apr 2022 15:50:00 +0300
Active Jobs:         scheduled-test-tests-27497570
Events:
  Type    Reason            Age                  From                Message
  ----    ------            ----                 ----                -------
  Normal  SuccessfulCreate  5m41s                cronjob-controller  Created job scheduled-test-tests-2749757
```

As mentioned above, we have a scheduled callback for launching our test.

## Getting Scheduled Test Results

Now we can check whether the test is executed every minute for the schedule we provided.

```sh
kubectl testkube get execution
```

Output:

```sh
  ID                       | NAME                | TYPE               | STATUS  | LABELS  
+--------------------------+---------------------+--------------------+---------+--------+
  6256c98f418062706814e1fc | scheduled-test      | postman/collection | passed  |         
  6256c953418062706814e1fa | scheduled-test      | postman/collection | passed  |         
  6256c91e418062706814e1f8 | scheduled-test      | postman/collection | passed  |         
  6256c8db418062706814e1f6 | scheduled-test      | postman/collection | passed  |         
  6256c89f418062706814e1f4 | scheduled-test      | postman/collection | passed  |         
  6256c885418062706814e1f2 | scheduled-test      | postman/collection | passed  |         
  6256c87e418062706814e1f0 | scheduled-test      | postman/collection | passed  | 
```

The test is successfully regulary executed.

## Create a Test Suite with a Schedule

Let's create a Test Suite with a required schedule using the Testkube CLI command:

```sh
cat test/suites/testsuite.json | kubectl testkube create testsuite --name scheduled-testsuite --schedule="*/1 * * * *"
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


TestSuite created scheduled-testsuite 🥇
```

We successfuly created a scheduled Test Suite and can view a list of the available Test Suites:

```sh
kubectl testkube get testsuites
```

Output:

```sh
  NAME                | DESCRIPTION            | STEPS | LABELS | SCHEDULE    | STATUS | EXECUTION ID  
+---------------------+------------------------+-------+--------+-------------+--------+--------------+
  scheduled-testsuite | Run test several times |     2 |        | */1 * * * * |        |    
```

The scheduled test suite was created, but it was not yet executed. 

## Run a Scheduled Test Suite

In order start the execution of the test suite on a defined schedule, we need to run it using Testkube CLI command:

```sh
kubectl testkube run testsuite scheduled-testsuite
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Name          : scheduled-testsuite
Status        : queued


Test Suite queued for execution

Use the following command to get test execution details:
$ kubectl testkube get tse 
```

Test suite was successfully scheduled for execution and put into the queue.
We will skip Cron job details, they are fully similar to test one described above.

## Getting Scheduled Test Suite Results

The test suite is executed every minute for the schedule we provided.

```sh
kubectl testkube get tse
```

Output:

```sh
  ID                       | TEST SUITE NAME     | EXECUTION NAME                             | STATUS | STEPS | LABELS  
+--------------------------+---------------------+--------------------------------------------+--------+-------+--------+
  6256ce3f418062706814e210 | scheduled-testsuite | scheduled-testsuite.abnormally-in-lark     | passed |     2 |
  6256ce04418062706814e20c | scheduled-testsuite | scheduled-testsuite.kindly-evolved-primate | passed |     2 |
  6256cdcc418062706814e208 | scheduled-testsuite | scheduled-testsuite.formerly-champion-dodo | passed |     2 |
```

The test suite is successfully executed according to the schedule set.
