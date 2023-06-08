# Running Test Suites

To run your Tests Suites, pass the `testsuites run` command with the test name to your `kubectl testkube` plugin. Test Suites are started asynchronously by default.

```sh
testkube run testsuite test-example
```

```sh title="Expected output:"
Name: test-example.fairly-humble-tick
Status: running

  STATUSES | STEP | IDS | ERRORS
+----------+------+-----+-------+



Use the following command to get test suite  execution details:
$ kubectl testkube get tse 61e1136165e59a3183465125


Use the following command to get test suite execution details:
$ kubectl testkube watch tse 61e1136165e59a3183465125
```

After the test is started, you can check the current status of the test with `tests execution EXECUTION_ID`.

## Running Testsuites Synchronously

You can start a testsuite synchronously by passing the `-f` flag (like --follow) to your command:

```sh
testkube run testsuite test-example -f
```

```sh title="Expected output:"

Name          : testsuite-parallel
Execution ID  : 63d3cd05c6768fc8b574e2e8
Execution name: ts-testsuite-parallel-19
Status        : running
Duration: 38.530756ms

  STATUSES                  | STEP                           | IDS                            | ERRORS      
----------------------------+--------------------------------+--------------------------------+-------------
  running, running, running | run:testkube/cli-test,         | 63d3cd05c6768fc8b574e2e9,      | "", "", ""  
                            | run:testkube/demo-test, delay  | 63d3cd05c6768fc8b574e2ea, ""   |             
                            | 1000ms                         |                                |             
  queued                    | delay 5000ms                   | ""                             | ""   

...

Id:       63d3cd05c6768fc8b574e2e8
Name:     ts-testsuite-parallel-19
Status:   running
Duration: 22.138s

Labels:   
  STATUSES               | STEP                           | IDS                            | ERRORS      
-------------------------+--------------------------------+--------------------------------+-------------
  passed, passed, passed | run:testkube/cli-test,         | 63d3cd05c6768fc8b574e2e9,      | "", "", ""  
                         | run:testkube/demo-test, delay  | 63d3cd05c6768fc8b574e2ea, ""   |             
                         | 1000ms                         |                                |             
  running                 | delay 5000ms                   | ""                             | ""  

...


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
$ kubectl testkube get tse 61e1142465e59a318346512b
``ì
