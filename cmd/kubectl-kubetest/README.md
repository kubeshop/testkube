# Kubectl kubetest plugin

## Installation 

TODO 

## Usage 

0) First you'll need to define test, tests are defined as Curstom Resource in Kubernetes cluster
   (access to Kubernetes cluster would be also needed)


You can create new test by applying 'by hand' CR e.g. 

```yaml
apiVersion: tests.kubetest.io/v1
kind: Script
metadata:
  name: test-kubeshop
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
```

Where content is simply exported postman collection in example above. 
Name is unique Sript Custom Resource name. 
Type is `postman/collection` as it runs exported postman collections.

If you don't want to create files we have a little helper for this: 

```sh
kubectl kubetest scripts create --file my_collection_file.json --name my-test-name
```



1) Starting new script execution 

```
$ kubectl kubetest scripts start my-test-name 

Script "my-test-name" started
Execution ID 02wi02-29329-2392930-93939

```

2) [TODO] Aborting already started script execution 
```
$ kubectl kubetest scripts abort SOME_EXECUTION_ID
Script "SCRIPTNAME" Execution aborted

```


3) Getting available scripts
```
$ kubectl kubetest scripts list

ID         NAME              Type
040-134   HomePage test      postman/collection   
123-246   Contact API test   postman/collection

```
 
4) Getting available executions

```sh
kubectl kubetest scripts executions

ID         NAME             Status     Complete   Start              End
1233-333   HomePage run     pending    75%        2021-07-30 12:33   
1233-332   HomePage run     pending    100%       2021-07-30 12:33   2021-07-30 13:10
```

5) Getting execution details

```sh
kubectl kubetest scripts execution test 6103a45b7e18c4ea04883866

....
some execution details
```







