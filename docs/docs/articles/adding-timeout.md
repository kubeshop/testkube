# Adding Timeouts to your Tests

<Tabs>
  <TabItem value="dashboard" label="Dashboard" default>
    Click on a Test > Settings > General > Timout

    ![How to setup Test Timeout in Dashboard](../img/dashboard-timeout.png)
  </TabItem>
  <TabItem value="cli" label="CLI">
    You can use the `--timemout` with the number of second in all commands related to a test. 

    ```sh
    testkube update test --name my-test --timeout 10
    ```
    ```sh title="Expected output:"
    Test updated testkube / curl 🥇
    ```
  </TabItem>
  <TabItem value="crd" label="Custom Resource">
    Add the following field to your Test CRD: 

    ```yaml
    executionRequest:
      activeDeadlineSeconds: 10
    ```

    A full CRD example here: 
    
    ```yaml
    apiVersion: tests.testkube.io/v3
    kind: Test
    metadata:
      name: my-test
      namespace: testkube
      labels:
        executor: curl-executor
        test-type: curl-test
    spec:
      type: curl/test
      content:
        executionRequest:
          activeDeadlineSeconds: 50
        type: string
        repository:
        data: "{\n  \"command\": [\n    \"curl\",\n    \"https://reqbin.com/echo/get/json\",\n    \"-H\",\n    \"'Accept: application/json'\"\n  ],\n  \"expected_status\": \"200\",\n  \"expected_body\": \"{\\\"success\\\":\\\"true\\\"}\"\n}"
    ```

  </TabItem>
</Tabs>