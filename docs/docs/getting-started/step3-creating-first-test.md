# Step 3 - Creating your first Test

## Using the CLI or the Dashboard

You can create your first test using the CLI or the Testkube Dashboard, both are great options!

To explore the Testkube dashboard, run the command:

```sh
testkube dashboard
```

## Kubernetes-native Tests

Tests in Testkube are created as a Custom Resources in Kubernetes and live inside your cluster.

You can create your tests directly as a Custom Resource, or use the CLI or the Testkube Dashboard to create them.

This section provides an example of creating a _Postman_ test. Nevertheless, Testkube supports a long [list of testing tools](../category/test-types).

## Creating a Postman test

First, let's create a `postman-collection.json` file containing a Postman test (the file content should look similar to the one below):

```json title="postman-collection.json"
{
  "info": {
    "_postman_id": "8af42c21-3e31-49c1-8b27-d6e60623a180",
    "name": "Kubeshop",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Home",
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Body matches string\", function () {",
              "    pm.expect(pm.response.text()).to.include(\"Accelerator\");",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ],
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "https://kubeshop.io/",
          "protocol": "https",
          "host": ["kubeshop", "io"],
          "path": [""]
        }
      },
      "response": []
    },
    {
      "name": "Team",
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test(\"Status code is 200\", function () {",
              "    pm.response.to.have.status(200);",
              "});"
            ],
            "type": "text/javascript"
          }
        }
      ],
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "https://kubeshop.io/our-team",
          "protocol": "https",
          "host": ["kubeshop", "io"],
          "path": ["our-team"]
        }
      },
      "response": []
    }
  ]
}
```

And create the test by running the command:

```sh
testkube create test --file postman_collection.json --type postman/collection --name my-first-test
```

:::note
This example is testing that the website https://kubeshop.io is returning a `200` status code. It just demostrates a way of creating a simple Postman test. In practice, you would test an internal Kubernetes service.
:::

## Starting a new Test Execution

When our test is defined as a Custom Resource we can now run it:

```sh
testkube run test my-first-test
```

```sh title="Expected output:"
Type:              postman/collection
Name:              my-first-test
Execution ID:      63f4d0910ca9ed26798741ca
Execution name:    my-frst-test-1
Execution number:  1
Status:            running
Start time:        2023-02-21 14:09:21.163713965 +0000 UTC
End time:          0001-01-01 00:00:00 +0000 UTC
Duration:

Test execution started
Watch test execution until complete:
$ testkube watch execution my-frst-test-1


Use following command to get test execution details:
$ testkube get execution my-frst-test-1
```

## Getting the result of a Test Execution

To see the result of a Test Execution, first you need to get the Test Execution list:

```sh
testkube get executions
```

```sh title=Expected output:"
  ID                       | NAME              | TEST NAME       | TYPE               | STATUS | LABELS
---------------------------+-------------------+-----------------+--------------------+--------+---------
  63f4d0910ca9ed26798741ca | my-first-test-1   | my-first-test   | postman/collection | passed |
                           |                   |                 |                    |        |
```

Copy the ID of the test and see the full details of the execution by running:

```sh
testkube get execution 63f4d0910ca9ed26798741ca
```

```sh title="Expected output:"
# ... test details
Test execution completed with success in 14.342s 🥇
```

## Changing the Output Format

For lists and details, you can use different output formats via the `--output` flag. The following formats are currently supported:

- `RAW` - Raw output from the given executor (e.g., for Postman collection, it's terminal text with colors and tables).
- `JSON` - Test run data are encoded in JSON.
- `GO` - For go-template formatting (like in Docker and Kubernetes), you'll need to add the `--go-template` flag with a custom format. The default is `{{ . | printf("%+v") }}`. This will help you check available fields.