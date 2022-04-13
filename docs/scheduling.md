# Testkube Test and Test suite scheduling

In order to allow to run tests and test suites on regular basis we support scheduling mechanism for these objects.
CRDs both for test and test suite contain a `schedule` field used to define rules for launching them in time.
We decided to use the same schedule data format, that is used to define Kubernetes Cron jobs (
check wikepedia Cron format for details https://en.wikipedia.org/wiki/Cron)

# Architecture behind scheduling

We decided to not reinvent any scheduling engine, but just to reuse existing one from Kubernetes Cron jobs.
In fact, for each scheduled test or test suite we create a special cron job from this template
https://github.com/kubeshop/helm-charts/blob/main/charts/testkube-api/cronjob_template.yml
Technically, it's just a callback to Testkube api server method launching either test or test suite execution.
So, it will work pretty similar to scheduled test and test suite executions done by external scheduling platform. 

# Create test with a schedule

Let's create a test with a required schedule using Testkube CLI command

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

We successfuly created a scheduled test, let's check a list of the available tests

```sh
kubectl testkube get tests
```

Output:

```sh
  NAME              | TYPE               | CREATED                       | LABELS | SCHEDULE    | STATUS | EXECUTION ID              
+-------------------+--------------------+-------------------------------+--------+-------------+--------+--------------------------+
  scheduled-test    | postman/collection | 2022-04-13 12:37:40 +0000 UTC |        | */1 * * * * |        |                           
```

As you can see the scheduled test was created, but it was not yet executed. 

# Run scheduled test

In order start execuction of the test on defined schedule we need to run it using Testkube CLI command

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

Use following command to get test execution details:
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

Cron job for this test was successfully created and test was executed
Let's check Cron job details

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

As it was mentioned above we have a scheduled callback for launching our test.

#  Getting scheduled test results

And we can check, if the test is executed every minute for schedule we provided.

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

And we can see above that test is successfully regulary executed.

# Create test suite with a schedule

Let's create a test suite with a required schedule using Testkube CLI command

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

We successfuly created a scheduled test suite, let's check a list of the available test suites

```sh
kubectl testkube get testsuites
```

Output:

```sh
  NAME                | DESCRIPTION            | STEPS | LABELS | SCHEDULE    | STATUS | EXECUTION ID  
+---------------------+------------------------+-------+--------+-------------+--------+--------------+
  scheduled-testsuite | Run test several times |     2 |        | */1 * * * * |        |    
```

As you can see the scheduled test suite was created, but it was not yet executed. 

# Run scheduled test suite

In order start execuction of the test suite on defined schedule we need to run it using Testkube CLI command

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

Use following command to get test execution details:
$ kubectl testkube get tse 
```

Test suite was successfully scheduled for execution and put into the queue.
We will skip Cron job details, they are fully similar to test one described above.

#  Getting scheduled test suite results

And we can check, if the test suite is executed every minute for schedule we provided.

```sh
kubectl testkube get tse
```

Output:

```sh
  ID                       | TEST SUITE NAME     | EXECUTION NAME                             | STATUS | STEPS  
+--------------------------+---------------------+--------------------------------------------+--------+-------+
  6256ce3f418062706814e210 | scheduled-testsuite | scheduled-testsuite.abnormally-in-lark     | passed |     2  
  6256ce04418062706814e20c | scheduled-testsuite | scheduled-testsuite.kindly-evolved-primate | passed |     2  
  6256cdcc418062706814e208 | scheduled-testsuite | scheduled-testsuite.formerly-champion-dodo | passed |     2
```

And we can see above that test suite is successfully regulary executed.
