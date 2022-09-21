---
sidebar_position: 2
sidebar_label: Running
---

# Running Test Suites

To run your Tests Suites, pass `testsuites run` command with the test name to your `kubectl testkube` plugin. Test Suites are started asynchronously by default.

```bash
kubectl testkube run testsuite test-example

Name: test-example.fairly-humble-tick
Status: pending

  STEP | STATUS | ID | ERROR
+------+--------+----+-------+



Use the following command to get test suite  execution details:
$ kubectl testkube get tse 61e1136165e59a3183465125


Use the following command to get test suite execution details:
$ kubectl testkube watch tse 61e1136165e59a3183465125
```

After the test is started, you can check the current status of the test with `tests execution EXECUTION_ID`.

## **Running Testsuites Synchronously**

You can start a testsuite synchronously by passing the `-f` flag (like --follow) to your command:

```bash
kubectl testkube run testsuite test-example -f

Name: test-example.equally-enabled-heron
Status: pending

  STEP | STATUS | ID | ERROR
+------+--------+----+-------+

...


             STEP            | STATUS  |            ID            | ERROR
+----------------------------+---------+--------------------------+-------+
  run test: testkube/test1 | success | 61e1142465e59a318346512d |


Name: test-example.equally-enabled-heron
Status: pending

             STEP            | STATUS  |            ID            | ERROR
+----------------------------+---------+--------------------------+-------+
  run test: testkube/test1 | success | 61e1142465e59a318346512d |
  delay 2000ms               | success |                          |


...


Name: test-example.equally-enabled-heron
Status: success

             STEP            | STATUS  |            ID            | ERROR
+----------------------------+---------+--------------------------+-------+
  run test: testkube/test1 | success | 61e1142465e59a318346512d |
  delay 2000ms               | success |                          |
  run test: testkube/test1 | success | 61e1142a65e59a318346512f |



Use the following command to get test execution details:
$ kubectl testkube get tse 61e1142465e59a318346512b

```
