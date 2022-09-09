---
sidebar_position: 10
sidebar_label: cURL
---
# cURL Commands

Testkube is able to run cURL commands as tests. There are 2 possibilities to validate the outputs of the cURL command:

- By using the status returned.
- By checking the body of the response.

Below is an example of how to format the tests:

```js
{
  "command": [
    "curl",
    "https://reqbin.com/echo/get/json",
    "-H",
    "'Accept: application/json'"
  ],
  "expected_status": "200",
  "expected_body": "{\"success\":\"true\"}"
}
```

The test Custom Resource Definition (CRD) should be created with the type `curl/test`.

## **Running Tests Using cURL Commands**

### **Creating and Running a cURL Test**

Save a test in a format as described above. In this example, it is `curl-test.json`.

Create the test by running `kubectl testkube create test --file curl-test.json --name curl-test --type "curl/test"`.

Check if the test was created using the command `kubectl testkube get tests`. The output will be similar to:


```bash
       NAME       |        TYPE         
+-----------------+--------------------+
  curl-test      | curl/test  
```

The test can be run using `kubectl testkube run test curl-test` which gives the output:

```bash
Type          : curl/test
Name          : curl-test
Execution ID  : 613a2d7056499e6e3d5b9c3e
Execution name: sadly-optimal-ram

Test queued for execution

Use the following command to get test execution details:
$ kubectl testkube get execution 613a2d7056499e6e3d5b9c3e

Or watch the script execution until complete:
$ kubectl testkube watch execution 613a2d7056499e6e3d5b9c3e
```

As seen above, results can be checked using `kubectl testkube get execution 613a2d7056499e6e3d5b9c3e`, where the id of the execution is unique for each execution. Ensure that the correct id is used. The output will look something like:

```bash
Name: painfully-super-colt,Status: success,Duration: 534ms

HTTP/2 200 
date: Thu, 09 Sep 2021 15:51:15 GMT
content-type: application/json
content-length: 19
last-modified: Thu, 09 Sep 2021 13:07:39 GMT
cache-control: max-age=31536000
cf-cache-status: HIT
age: 6939
accept-ranges: bytes
expect-ct: max-age=604800, report-uri="https://report-uri.cloudflare.com/cdn-cgi/beacon/expect-ct"
report-to: {"endpoints":[{"url":"https:\/\/a.nel.cloudflare.com\/report\/v3?s=OZHPfvLjuVhpklzeGvhs8Ic0w%2FJ1%2BKgMcXeichnmMt9lKxF%2Fkco%2FHD2Z2vWfvInH9IPNuAQpjKu1Roqy8efIhVztIhvBP14Wx4wdBsQhzxUe9znZ%2Fmanwsky5G3Q"}],"group":"cf-nel","max_age":604800}
nel: {"success_fraction":0,"report_to":"cf-nel","max_age":604800}
server: cloudflare
cf-ray: 68c193af1f706571-LHR
alt-svc: h3=":443"; ma=86400, h3-29=":443"; ma=86400, h3-28=":443"; ma=86400, h3-27=":443"; ma=86400

{"success":"true"}
```

There is a generated name for the execution, the status, duration and the output of the cURL command.
