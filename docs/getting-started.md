# Getting Started 

Please follow [install steps](/docs/installing.md) for kubetest installation

## Getting help 

```sh 
kubectl kubetest --help 

# or for scripts runs
kubectl kubetest scripts --help 
```
## Defining tests

First you'll need to define test, tests are defined as Curstom Resource in Kubernetes cluster (access to Kubernetes cluster would be also needed)

For now we're handling exported *Postman collections* but in future we plan to handle more testing tools.

If you don't want to create Custom Resources "by hand" we have a little helper for this: 

```sh
kubectl kubetest scripts create --file my_collection_file.json --name my-test-name

#or 
cat my_collection_file.json | kubectl kubetest scripts create --name my-test-name
```


You can also create new test by creating new Custom Resource 'by hand' (via `kubectl apply -f ...`) it's equivalent to running `kubectl kubetest create` command e.g.:

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: tests.kubetest.io/v1
kind: Script
metadata:
  name: my-test-name
spec:
  # Add fields here
  id: Some internal ID 
  type: postman/collection
  content: >
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
                  "    pm.expect(pm.response.text()).to.include(\"K8s Accelerator\");",
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
              "host": [
                "kubeshop",
                "io"
              ],
              "path": [
                ""
              ]
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
                  "});",
                  "",
                  "pm.test(\"Body matches string\", function () {",
                  "    pm.expect(pm.response.text()).to.include(\"Jacek Wysocki\");",
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
              "host": [
                "kubeshop",
                "io"
              ],
              "path": [
                "our-team"
              ]
            }
          },
          "response": []
        }
      ]
    }
EOF
```

Where:
- `content` is exported postman collection in example above. 
- `name` is unique Sript Custom Resource name. 
- `type` is `postman/collection` as it runs exported postman collections.

## Starting new script execution 

When our script is defined as CR we can now run it: 

```
$ kubectl kubetest scripts start my-test-name 

... some script run data ...

Script execution completed 

Use following command to get script execution details:
$ kubectl kubetest scripts execution test 611b6da38cd74034e7c9d408
```

## Getting execution details
After script completed with success or error you can go back to script details by running 
scripts execution command:

```sh
kubectl kubetest scripts execution script-name 6103a45b7e18c4ea04883866

....
some execution details
```



## Getting available scripts

To run script execution you'll need to know script name

```
$ kubectl kubetest scripts list

+----------------------+--------------------+
|         NAME         |        TYPE        |
+----------------------+--------------------+
| create-001-test      | postman/collection |
| envs1                | postman/collection |
| test-kubeshop        | postman/collection |
| test-kubeshop-failed | postman/collection |
| test-postman-script  | postman/collection |
+----------------------+--------------------+

```
 
## Getting available executions

```sh
kubectl kubetest scripts executions script-name

+------------+--------------------+--------------------------+---------------------------+----------+
|   SCRIPT   |        TYPE        |       EXECUTION ID       |      EXECUTION NAME       | STATUS   |
+------------+--------------------+--------------------------+---------------------------+----------+
| parms-test | postman/collection | 611a5a1a910ca385751eb2c6 | pt1                       | success  |
| parms-test | postman/collection | 611a5a40910ca385751eb2c8 | pt2                       | error    |
| parms-test | postman/collection | 611a5b6d910ca385751eb2ca | forcibly-frank-panda      | error    |
| parms-test | postman/collection | 611a5b83910ca385751eb2cc | slightly-merry-jennet     | error    |
| parms-test | postman/collection | 611b6d6c8cd74034e7c9d406 | frequently-expert-terrier | error    |
| parms-test | postman/collection | 611b6da38cd74034e7c9d408 | violently-fresh-elephant  | error    |
+------------+--------------------+--------------------------+---------------------------+----------+
```

## [TODO] Aborting already started script execution - NOT IMPLEMENTED
```
$ kubectl kubetest scripts abort SOME_EXECUTION_ID
Script "SCRIPTNAME" Execution aborted

```


## Changing output format

For lists and details you can use different ouput format (`--output` flag) for now we're supporting following formats:
- RAW - it's raw output from given executor (e.g. for Postman collection it's terminal text with colors and tables)
- JSON - test run data are encoded in JSON 
- GO - go-template formatting (like in Docker and Kubernetes) you'll need to add `--go-template` flag with custom format (defaults to {{ . | printf("%+v") }} will help you check available fields) 

(Keep in mind that we have plans for other output formats like junit etc.)
