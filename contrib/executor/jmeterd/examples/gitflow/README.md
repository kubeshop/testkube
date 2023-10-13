# GitFlow Example test for Distributed JMeter

This test is an example of how to run a distributed JMeter test using a git repo as a source and how to use the advanced features of the executor.

## Test Breakdown

### Plugins
All the plugins required by the test are kept in the `plugins` directory of the test folder in the git repo.

### Additional Files
* **CSV**: The test references a CSV file named `Credentials.csv` located in the `data/` directory relative to the project home directory (`${PROJECT_HOME}`). 
  This CSV should contain columns `USERNAME` and `PASSWORD`.

### Environment Variables
* **DATA_CONFIG**: Used to determine the directory of the CSV data file. It defaults to `${PROJECT_HOME}` if not provided.

### Properties
* **JMETER_UC1_NBUSERS**: Number of users for the test. Defaults to `2` if not provided.
* **JMETER_UC1_RAMPUP**: Ramp-up period for the test in seconds. Defaults to `2` if not provided.
* **JMETER_URI_PATH**: The URI path to test against. Defaults to `/pricing` if not provided.

## Steps to execute this Test

### Testkube Dashboard
1. Open the Testkube Dashboard and create a new test.
2. Type a test name (i.e. `jmeterd-example`) and select `jmeterd/test` as test type.
3. Select `Git` as the source type and fill the following details:
   * Git Repository URI: https://github.com/kubeshop/testkube
   * Branch: develop
   * Path: contrib/executor/jmeterd/examples/gitflow
4. Click **Create** to create the test.
5. Select **Settings** tab and then open the **Variables & Secrets** tab from the left menu.
6. Add a new variable called **SLAVES_COUNT** and set it to the number of slave pods you want to spawn for the test.
7. Add another variable called **DATA_CONFIG** and set it to `/data/repo/contrib/executor/jmeterd/examples/gitflow`
8. Click on **Save** underneath **the Variables & Secrets** section.
9. In the Arguments section, set the following arguments: `-GJMETER_UC1_NBUSERS=5 jmeter-properties-external.jmx`
10. Click on **Save** underneath the **Arguments** section.
11. Run the test

### CRD

You can also apply the following Test CRD to create the example test:
```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: jmeterd-example
  namespace: testkube
  labels:
    executor: jmeterd-executor
    test-type: jmeterd-test
spec:
  type: jmeterd/test
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube
      branch: develop
      path: contrib/executor/jmeterd/examples/gitflow
  executionRequest:
    variables:
      DATA_CONFIG:
        name: DATA_CONFIG
        value: "/data/repo/contrib/executor/jmeterd/examples/gitflow"
        type: basic
      SLAVES_COUNT:
        name: SLAVES_COUNT
        value: "2"
        type: basic
    args:
      - "-GJMETER_UC1_NBUSERS=5"
      - "jmeter-properties-external.jmx"
```