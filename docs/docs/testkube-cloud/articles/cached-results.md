# Cached Test Results

Testkube cached test results allows you to see and inspect test execution results even when your Testkube agent is offline.

## Overview

![offline-main](../../img/offline-list.png)

Testkube Cloud uses test execution data stored in Cloud to allow you inspect past test executions. This feature also works when your agent is online, but the Testkube agent doesn't have the test definition available in Kubernetes.

Cached test results appear with a read-only tag. These tests cannot be updated. If you want to get rid of old tests, you can go to the Test Settings page and click "Delete Test".
