import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";

# Test Workflows - Matrix and Sharding

Often you want to run a test with multiple scenarios or environments,
either to distribute the load or to verify it on different setup.

Test Workflows have a built-in mechanism for all these cases - both static and dynamic.

## Usage

Matrix and sharding features are supported in [**Services (`services`)**](./test-workflows-services.md), and both [**Test Suite (`execute`)**](./test-workflows-test-suites.md) and [**Parallel Steps (`parallel`)**](./test-workflows-parallel.md) operations.

<Tabs>
<TabItem value="services" label={<span>Services (<code>services</code>)</span>} default>

```yaml
kind: TestWorkflow
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: example-matrix-services
spec:
  services:
    remote:
      matrix:
        browser:
        - driver: chrome
          image: selenium/standalone-chrome:4.21.0-20240517
        - driver: edge
          image: selenium/standalone-edge:4.21.0-20240517
        - driver: firefox
          image: selenium/standalone-firefox:4.21.0-20240517
      image: "{{ matrix.browser.image }}"
      description: "{{ matrix.browser.driver }}"
      readinessProbe:
        httpGet:
          path: /wd/hub/status
          port: 4444
        periodSeconds: 1
  steps:
  - shell: 'echo {{ shellquote(join(map(services.remote, "tojson(_.value)"), "\n")) }}'
```

</TabItem>
<TabItem value="execute" label={<span>Test Suite (<code>execute</code>)</span>}>

```yaml
kind: TestWorkflow
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: example-matrix-test-suite
spec:
  steps:
  - execute:
      workflows:
      - name: k6-workflow-smoke
        matrix:
          target:
          - https://testkube.io
          - https://docs.testkube.io
        config:
          target: "{{ matrix.target }}"
```

</TabItem>
<TabItem value="parallel" label={<span>Parallel Steps (<code>parallel</code>)</span>}>

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: example-sharded-playwright
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube
      paths:
      - test/playwright/executor-tests/playwright-project
  container:
    image: mcr.microsoft.com/playwright:v1.32.3-focal
    workingDir: /data/repo/test/playwright/executor-tests/playwright-project

  steps:
  - name: Install dependencies
    shell: 'npm ci'

  - name: Run tests
    parallel:
      count: 2
      transfer:
      - from: /data/repo
      shell: 'npx playwright test --shard {{ index + 1 }}/{{ count }}'
```

</TabItem>
</Tabs>

## Syntax

This feature allows you to provide few properties:

* `matrix` to run the operation for different combinations
* `count`/`maxCount` to replicate or distribute the operation
* `shards` to provide the dataset to distribute among replicas

Both `matrix` and `shards` can be used together - all the sharding (`shards` + `count`/`maxCount`) will be replicated for each `matrix` combination.

### Matrix

Matrix allows you to run the operation for multiple combinations. The values for each instance are accessible by `matrix.<key>`.

In example:

```yaml
parallel:
  matrix:
    image: ['node:20', 'node:21', 'node:22']
    memory: ['1Gi', '2Gi']
  container:
    resources:
      requests:
        memory: '{{ matrix.memory }}'
  run:
    image: '{{ matrix.image }}'
```

Will instantiate 6 copies:

| `index` | `matrixIndex` | `matrix.image` | `matrix.memory` | `shardIndex` |
|---------|---------------|----------------|-----------------|--------------|
| `0`     | `0`           | `"node:20"`    | `"1Gi"`         | `0`          |
| `1`     | `1`           | `"node:20"`    | `"2Gi"`         | `0`          |
| `2`     | `2`           | `"node:21"`    | `"1Gi"`         | `0`          |
| `3`     | `3`           | `"node:21"`    | `"2Gi"`         | `0`          |
| `4`     | `4`           | `"node:22"`    | `"1Gi"`         | `0`          |
| `5`     | `5`           | `"node:22"`    | `"2Gi"`         | `0`          |

The matrix properties can be a static list of values, like:

```yaml
matrix:
  browser: [ 'chrome', 'firefox', '{{ config.another }}' ]
```

or could be dynamic one, using [**Test Workflow's expressions**](test-workflows-expressions.md):

```yaml
matrix:
  files: 'glob("/data/repo/**/*.test.js")'
```

### Sharding

Often you may want to distribute the load, to speed up the execution. To do so, you can use `shards` and `count`/`maxCount` properties.

* `shards` is a map of data to split across different instances
* `count`/`maxCount` are describing the number of instances to start
  * `count` defines static number of instances (always)
  * `maxCount` defines maximum number of instances (will be lower if there is not enough data in `shards` to split)

<Tabs>
<TabItem value="count" label={<span>Replicas (<code>count</code> only)</span>} default>

```yaml
parallel:
  count: 5
  description: "{{ index + 1 }} instance of {{ count }}"
  run:
    image: grafana/k6:latest
```
__
</TabItem>
<TabItem value="count-shard" label={<span>Static sharding (<code>count</code> + <code>shards</code>)</span>} default>

```yaml
parallel:
  count: 2
  description: "{{ index + 1 }} instance of {{ count }}"
  shards:
    url: ["https://testkube.io", "https://docs.testkube.io", "https://app.testkube.io"]
  run:
    # shard.url for 1st instance == ["https://testkube.io", "https://docs.testkube.io"]
    # shard.url for 2nd instance == ["https://app.testkube.io"]
    shell: 'echo {{ shellquote(join(shard.url, "\n")) }}'
```

</TabItem>
<TabItem value="max-count-shard" label={<span>Dynamic sharding (<code>maxCount</code> + <code>shards</code>)</span>} default>

```yaml
parallel:
  maxCount: 5
  shards:
    # when there will be less than 5 tests found - it will be 1 instance per 1 test
    # when there will be more than 5 tests found - they will be distributed similarly to static sharding
    testFiles: 'glob("cypress/e2e/**/*.js")'
  description: '{{ join(map(shard.testFiles, "relpath(_.value, \"cypress/e2e\")"), ", ") }}'
```

</TabItem>
</Tabs>

Similarly to `matrix`, the `shards` may contain a static list, or [**Test Workflow's expression**](test-workflows-expressions.md).

### Counters

Besides having the `matrix.<key>` and `shard.<key>` there are some counter variables available in Test Workflow's expressions:

* `index` and `count` - counters for total instances
* `matrixIndex` and `matrixCount` - counters for the combinations
* `shardIndex` and `shardCount` - counters for the shards

### Matrix and sharding together

Sharding can be run along with matrix. In that case, for every matrix combination, we do have selected replicas/sharding. In example:

```yaml
matrix:
  browser: ["chrome", "firefox"]
  memory: ["1Gi", "2Gi"]
count: 2
shards:
  url: ["https://testkube.io", "https://docs.testkube.io", "https://app.testkube.io"]
```

Will start 8 instances:

| `index` | `matrixIndex` | `matrix.browser` | `matrix.memory` | `shardIndex` | `shard.url`                                           |
|---------|---------------|------------------|-----------------|--------------|-------------------------------------------------------|
| `0`     | `0`           | `"chrome"`       | `"1Gi"`         | `0`          | `["https://testkube.io", "https://docs.testkube.io"]` |
| `1`     | `0`           | `"chrome"`       | `"1Gi"`         | `1`          | `["https://app.testkube.io"]`                         |
| `2`     | `1`           | `"chrome"`       | `"2Gi"`         | `0`          | `["https://testkube.io", "https://docs.testkube.io"]` |
| `3`     | `1`           | `"chrome"`       | `"2Gi"`         | `1`          | `["https://app.testkube.io"]`                         |
| `4`     | `2`           | `"firefox"`      | `"1Gi"`         | `0`          | `["https://testkube.io", "https://docs.testkube.io"]` |
| `5`     | `2`           | `"firefox"`      | `"1Gi"`         | `1`          | `["https://app.testkube.io"]`                         |
| `6`     | `3`           | `"firefox"`      | `"2Gi"`         | `0`          | `["https://testkube.io", "https://docs.testkube.io"]` |
| `7`     | `3`           | `"firefox"`      | `"2Gi"`         | `1`          | `["https://app.testkube.io"]`                         |
