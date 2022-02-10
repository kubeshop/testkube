# Testkube test tests

Test tests are single executor oriented tests. Test can have different types, which depends what executors are installed in your cluster.

Testkubes includes `postman/collection`, `cypress/project` and `curl/test` test types which are auto registered during testkube install by default. // provide examples for cypress and  curl

As Testkube was designed with flexibility in mind - you can add your own executor which will handle additional test types.

## Test test source

Tests can be currently created from two sources:

- First one is simple `file` with test content e.g. for postman collections we're exporting collection as json file, or for curl executor we're passing json file with configured curl command.
- Second source handled by Testkube is `git` - we can pass `repository`, `path` and `branch` where our tests is. This one is used in Cypress executor - as Cypress tests are more like npm-based projects which can have a lot of files. We're handling here sparse checkouts which are fast even in case of huge mono-repos

## Create test

### Create your first test from file (Postman Collection test)

To create your first postman collection in Testkube you'll first need to export your collection into a file

Right click on your collection name
![create postman colletion step 1](img/test-create-1.png)
Click "Export" button
![create postman colletion step 2](img/test-create-1.png)
Save in convinient location (We're using `~/Downloads/TODO.postman_collection.json` path)
![create postman colletion step 3](img/test-create-1.png)

```sh
kubectl testkube tests create --file ~/Downloads/TODO.postman_collection.json --name test
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

Test created! Now we can run as many times as we want.

### Updating tests

If you need to update your test after change in Postman just re-export it to file and run update command:

```sh
kubectl testkube tests update --file ~/Downloads/TODO.postman_collection.json --name test
```

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

Testkube will override all test settings and content with `update` method.

### Checking tests content

Let's see what has been created:

```sh
kubectl get tests -n testkube
```

Output:

```sh
NAME   AGE
test   32s
```

```sh
$ kubectl testkube tests get test

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

We can see that test resource was created with Postman collection JSON content.

You can also check tests with standard `kubectl` comman which will list Tests Custom Resource

```sh
kubectl get tests -ntestkube test -oyaml
```

### Create test from git

Some executors can handle files and some could handle only git resources - You'll need to follow particular executor readme file to be aware what tests type does he handle.

Let's assume that some Cypress project is created in some git repository - Let's assume we've created one in Let's assume we've created one in <https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples>  

Where `examples` is test directory in `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress based test (in git based tests we need to pass test type)

```sh
kubectl testkube tests create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
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

Let's check how created testkube test test is defined on cluster: 

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

As we can see this test has `spec.repository` with git repository data. Those data can now be used by executor to download test data.

## Summary

Tests are main smallest abstractions over tests in Testkube, they can be created with different sources and used by executors to run on top of particular test framework.
