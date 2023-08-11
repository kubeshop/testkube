# Test and Test Suite Execution CRDs

Testkube allows you to automatically run tests and test suites by creating or updating Test or Test Suite Execution CRDs.

## What are Testkube Execution CRDs?

In generic terms, an _Execution_ defines a _test_ or _testsuite_ which will be executed when CRD is created or updated. For example, we could define a _TestExecution_ which _runs_ a _Test_ when a _TestExecution_ gets _modified_.

#### Selecting Resource

Names are used when we want to select a specific resource. 

```yaml
test:
  name: Testkube test name
```

or 

```yaml
testSuite:
  name: Testkube test suite name
```

### Execution Request

An Execution Request defines execution parameters for each specific resource.

## Example

Here are examples for a **Test Execution** *testexecution-example* which runs the **Test** *test-example*
when a **Test Execution** is created or updated and a **Test Suite Execution** *testsuiteexecution-example* 
which runs the **Test Suite** *testsuite-example * when a **Test Suite Execution** is created or updated.

```yaml
apiVersion: tests.testkube.io/v1
kind: TestExecution
metadata:
  name: testexecution-example
spec:
  test:
    name: test-example
  executionRequest:
    variables:
      VAR_TEST:
        name: VAR_TEST
        value: "ANY"
        type: basic
```

```yaml
apiVersion: tests.testkube.io/v1
kind: TestSuiteExecution
metadata:
  name: testsuiteexecution-example
spec:
  testSuite:
    name: testsuite-example
  executionRequest:
    variables:
      VAR_TEST:
        name: VAR_TEST
        value: "ANY"
        type: basic
```

## Architecture

Testkube uses a Kubernetes Operator to reconcile Test and Test Suite Execution CRDs state and run the corresponding test and test suite when resource generation is changed.