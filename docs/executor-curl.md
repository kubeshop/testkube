# Curl Commands

Testkube is able to run curl commands as tests, there are 2 possibilities to validate the outputs of the curl command, one using the status returned and the other checking the body of the response. Bellow is an example on how to format the tests.

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

The test CRD should be created with the type `curl/test`.

## Running tests using curl commands

### Creating and running a curl test

Save a test in a format as described above into let's say `curl-test.json`

Create the test by running `kubectl testkube tests create --file curl-test.json --name curl-test --type "curl/test"`

Check if it was created using command `kubectl testkube tests list` it will output something like:

```sh
       NAME       |        TYPE         
+-----------------+--------------------+
  curl-test      | curl/test  
```

Test can be run using `kubectl testkube tests start curl-test` which gives the output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop

Test queued for execution

Use following command to get test execution details:
$ kubectl testkube tests execution 613a2d7056499e6e3d5b9c3e

or watch test execution until complete:
$ kubectl testkube tests watch 613a2d7056499e6e3d5b9c3e
```

As in the output is stated results can be checked using `kubectl testkube tests execution 613a2d7056499e6e3d5b9c3e` where the id of the execution is unique for each execution, make sure that the right id is used. Output of that should look something like:

```sh
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

where there is a generated name for the execution, the status, duration and the output of the curl command.
