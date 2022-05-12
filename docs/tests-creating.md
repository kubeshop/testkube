# Testkube Tests

Tests are single executor oriented objects. Test can have different types, which depends on which executors are installed in your cluster.

Testkube includes `postman/collection`, `cypress/project`, `curl/test`, `k6/script` and `soapui/xml` test types which are auto registered during the Testkube install by default.

As Testkube was designed with flexibility in mind, you can add your own executors to handle additional test types.

## **Test Source**

Tests can be currently created from multiple sources:

1. A simple `file` with the test content, For example, with Postman collections, we're exporting the collection as a JSON file. For cURL executors, we're passing a JSON file with the configured cURL command.
2. String - we can also define the content of the test as a string
3. Git directory - we can pass `repository`, `path` and `branch` where our tests are stored. This is used in Cypress executor as Cypress tests are more like npm-based projects which can have a lot of files. We are handling sparse checkouts which are fast even in the case of huge mono-repos.
4. Git file - similarly to Git directories, we can use files located on Git by specifying `git-uri` and `branch`.

Note: not all executors support all input types. Please refer to the individual executors' documentation to see which options are available.

## **Create a Test**

### **Create Your First Test from a File (Postman Collection Test)**

To create your first Postman collection in Testkube, export your collection into a file.

Right click on your collection name:

![create postman collection step 1](img/test-create-1.png)

Click the **Export** button:

![create postman collection step 2](img/test-create-2.png)

Save in a convenient location. In this example, we are using `~/Downloads/TODO.postman_collection.json` path.

![create postman collection step 3](img/test-create-3.png)

Create a Testkube test using the exported JSON and give it a unique and fitting name. For simplicity's sake we used `test` in this example.

```sh
kubectl testkube create test --file ~/Downloads/TODO.postman_collection.json --name test
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████
                                           /tɛst kjub/ by Kubeshop


Detected test type postman/collection
Test created test 🥇
```

Test created! Now we have a reusable test.

### **Updating Tests**

If you need to update your test after change in Postman, re-export it to a file and run the update command:

```sh
kubectl testkube update test --file ~/Downloads/TODO.postman_collection.json --name test
```

To check if the test was created correctly, look at the `Test Custom Resource` in your Kubernetes cluster: 

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████
                                           /tɛst kjub/ by Kubeshop


Detected test test type postman/collection
Test updated test 🥇
```

Testkube will override all test settings and content with the `update` method.

### **Checking Test Content**

Let's see what has been created:

```sh
kubectl get tests -ntestkube
```

Output:

```sh
NAME   AGE
test   32s
```

Get the details of a test: 

```sh 
kubectl get tests -ntestkube test-example -oyaml
```sh
$ kubectl testkube get test test

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████
                                           /tɛst kjub/ by Kubeshop


name: test
type_: postman/collection
content: |-
    {
        "info": {
                "_postman_id": "b40de9fe-9201-4b03-8ca2-3064d9027dd6",
                "name": "TODO",
                "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": [
                {
                        "name": "Create TODO",
                        "event": [
                                {
                                        "listen": "test",
                                        "script": {
                                                "exec": [
                                                        "pm.test(\"Status code is 201 CREATED\", function () {",
                                                        "    pm.response.to.have.status(201);",
                                                        "});",
                                                        "",
                                                        "",
                                                        "pm.test(\"Check if todo item craeted successfully\", function() {",
                                                        "    var json = pm.response.json();",
                                                        "    pm.environment.set(\"item\", json.url);",
                                                        "    pm.sendRequest(json.url, function (err, response) {",
                                                        "        var json = pm.response.json();",
                                                        "        pm.expect(json.title).to.eq(\"Create video for conference\");",
                                                        "",
                                                        "    });",
                                                        "    console.log(\"creating\", pm.environment.get(\"item\"))",
                                                        "})",
                                                        "",
                                                        ""
                                                ],
                                                "type": "text/javascript"
                                        }
                                },
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        ""
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "protocolProfileBehavior": {
                                "disabledSystemHeaders": {}
                        },
                        "request": {
                                "method": "POST",
                                "header": [
                                        {
                                                "key": "Content-Type",
                                                "value": "application/json",
                                                "type": "text"
                                        }
                                ],
                                "body": {
                                        "mode": "raw",
                                        "raw": "{\"title\":\"Create video for conference\",\"order\":1,\"completed\":false}"
                                },
                                "url": {
                                        "raw": "{{uri}}",
                                        "host": [
                                                "{{uri}}"
                                        ]
                                }
                        },
                        "response": []
                },
                {
                        "name": "Complete TODO item",
                        "event": [
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        "console.log(\"completing\", pm.environment.get(\"item\"))"
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "request": {
                                "method": "PATCH",
                                "header": [
                                        {
                                                "key": "Content-Type",
                                                "value": "application/json",
                                                "type": "text"
                                        }
                                ],
                                "body": {
                                        "mode": "raw",
                                        "raw": "{\"completed\": true}"
                                },
                                "url": {
                                        "raw": "{{item}}",
                                        "host": [
                                                "{{item}}"
                                        ]
                                }
                        },
                        "response": []
                },
                {
                        "name": "Delete TODO item",
                        "event": [
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        "console.log(\"deleting\", pm.environment.get(\"item\"))"
                                                ],
                                                "type": "text/javatest"
                                        }
                                },
                                {
                                        "listen": "test",
                                        "script": {
                                                "exec": [
                                                        "pm.test(\"Status code is 204 no content\", function () {",
                                                        "    pm.response.to.have.status(204);",
                                                        "});"
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "request": {
                                "method": "DELETE",
                                "header": [],
                                "url": {
                                        "raw": "{{item}}",
                                        "host": [
                                                "{{item}}"
                                        ]
                                }
                        },
                        "response": []
                }
        ],
        "event": [
                {
                        "listen": "prerequest",
                        "script": {
                                "type": "text/javascript",
                                "exec": [
                                        ""
                                ]
                        }
                },
                {
                        "listen": "test",
                        "script": {
                                "type": "text/javascript",
                                "exec": [
                                        ""
                                ]
                        }
                }
        ],
        "variable": [
                {
                        "key": "uri",
                        "value": "http://34.74.127.60:8080/todos"
                },
                {
                        "key": "item",
                        "value": null
                }
        ]
    }

```

We can see that the test resource was created with Postman collection JSON content.

You can also check tests with the standard `kubectl` command which will list the tests Custom Resource.

```sh
kubectl get tests -n testkube test -oyaml
```

### **Create a Test from Git**

Some executors can handle files and some can handle only git resources. You'll need to follow the particular executor **readme** file to be aware which test types the executor handles.

Let's assume that a Cypress project is created in a git repository - <https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples> - where **examples** is a test directory in the `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress-based test as shown below. In git based tests, we need to pass the test type.

```sh
kubectl testkube create test --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████
                                           /tɛst kjub/ by Kubeshop


Test created kubeshop-cypress 🥇
```

Let's check how the test created by Testkube is defined in the cluster:

```sh
$ kubectl get tests -n testkube kubeshop-cypress -o yaml
apiVersion: tests.testkube.io/v1
kind: Test
metadata:
  creationTimestamp: "2021-11-17T12:29:32Z"
  generation: 1
  name: kubeshop-cypress
  namespace: testkube
  resourceVersion: "225162"
  uid: f0d856aa-04fc-4238-bb4c-156ff82b4741
spec:
  repository:
    branch: main
    path: examples
    type: git
    uri: https://github.com/kubeshop/testkube-executor-cypress.git
  type: cypress/project
```

As we can see, this test has `spec.repository` with git repository data. This data can now be used by the executor to download test data.

## **Summary**

Tests are the main smallest abstractions over test suites in Testkube, they can be created with different sources and used by executors to run on top of a particular test framework.
