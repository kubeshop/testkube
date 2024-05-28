# Test, Test Suite and Test Workflow Execution CRDs

Testkube allows you to automatically run tests, test suites and test workflows by creating or updating Test, Test Suite or Test Workflow Execution CRDs.

## What are Testkube Execution CRDs?

In generic terms, an _Execution_ defines a _test_, _testsuite_ or _testworkflow_ which will be executed when CRD is created or updated. For example, we could define a _TestExecution_ which _runs_ a _Test_ when a _TestExecution_ gets _modified_.

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

or 

```yaml
testWorkflow:
  name: Testkube test workflow name
```

### Execution Request

An Execution Request defines execution parameters for each specific resource.

## Example

Here are examples for a **Test Execution** *testexecution-example* which runs the **Test** *test-example*
when a **Test Execution** is created or updated, a **Test Suite Execution** *testsuiteexecution-example* 
which runs the **Test Suite** *testsuite-example* when a **Test Suite Execution** is created or updated
and **Test Workflow Execution** *testworkflowexecution-example* which runs the **Test Workflow** *testworkflow-example*
when a **Test Workflow Execution** is created or updated

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

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflowExecution
metadata:
  name: testworkflowexecution-example
spec:
  testWorkflow:
    name: testworkflow-example
  executionRequest:
    config:
      browser: "chrome"
```

## Architecture

Testkube uses a Kubernetes Operator to reconcile Test, Test Suite and Test Workflow Execution CRDs state and run the corresponding test, test suite and test workflow when resource generation is changed.