# Cached Test & Test Suite Results

export const ProBadge = () => {
  return (
    <span>
      <p class="pro-badge">PRO FEATURE</p>
    </span>
  );
}

<ProBadge />

Testkube cached test results allows you to see and inspect test execution results even when your Testkube agent is offline.

## Overview

![offline-test](../../img/offline-list.png)

Testkube Cloud uses test execution data stored in Cloud to allow you inspect past test executions. This feature also works when your agent is online, but the Testkube agent doesn't have the test definition available in Kubernetes.

Cached test results appear with a read-only tag. These tests cannot be updated. If you want to get rid of old tests, you can go to the Test Settings page and click "Delete Test".

![offline-test-suite](../../img/offline-test-suite.png)

Similar to tests, Testkube Cloud supports also cached test suites, using the data stored in Cloud. These can be identified by the read-only tag which suggests that either your agent is not connected, or that a particular test suite definition is no longer available in Kubernetes.

## CLI

You can use Testkube CLI to retrieve read-only tests and test suites, their executions, and download artifacts. Make sure to use version v1.16.7 or greater. You can use the `testkube version` command to check your client version.

For example, listing tests:

```sh
testkube get tests
```

```yaml
  NAME   | DESCRIPTION | TYPE      | CREATED                       | LABELS                         | SCHEDULE | STATUS | EXECUTION ID       
---------+-------------+-----------+-------------------------------+--------------------------------+----------+--------+---------------------------
  k6 |             | k6/script | 0001-01-01 00:00:00 +0000 UTC | executor=k6-executor,          |          | failed | 64e4ace7dfca3109c5d2cc38
         |             |           |                               | test-type=k6-script            |          |        |                    
```

You can also list and get executions as well as download artifacts:

```sh
testkube get executions

testkube get execution 64e4ace7ca80a3290a4a762f

tk get artifact 654b867e234f24e69172b2ab
tk download artifacts 654b867e234f24e69172b2ab
```
