![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to testkube Executor Curl

testkube Executor Curl is the test executor for [testkube](https://testkube.io) that is using [Curl](https://curl.se/).

# Issues and enchancements

Please follow to main testkube repository for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

## Details

Curl executor is a very simple one, it runs a curl command given as the input and check the response for expected status and body, the input is of form

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

Templates can be used to parametrize input

```js
{
  "command": [
    "curl",
    "{{.url}}",
    "-H",
    "'{{.header}}'"
  ],
  "expected_status": "{{.status}}",
  "expected_body": "{{.body}}"
}
```

and the parameters will be passed by testkube using param flag ```--param key=value```

the executor will check if the response has `expected_status` and if body of the response contains the `expected_body`.

The type of the test CRD should be `curl/test`.

## API

testkube Executor Curl implements [testkube OpenAPI for executors](https://docs.testkube.io/openapi) (look at executor tag)
