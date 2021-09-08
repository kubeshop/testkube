# Curl Commands

Kubtest is able to run curl commands as tests, there are 2 possibilities to validate the outputs of the curl command, one using the status returned and the other checking the body of the response. Bellow is an example on how to format the tests.

```js
{
  "command": [
    "curl",
    "https://reqbin.com/echo/get/json",
    "-H",
    "'Accept: application/json'"
  ],
  "expected_status": 200,
  "expected_body": "{\"success\":\"true\"}"
}
```

The test CRD should be created with the type `curl/test`.
