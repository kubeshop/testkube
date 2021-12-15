# TestKube test scripts

Test scripts are single executor oriented tests. Script can have different types, which depends what executors are installed in your cluster.

TestKubes includes `postman/collection`, `cypress/project` and `curl/test` script types which are auto registered during testkube install by default. // provide examples for cypress and  curl

As TestKube was designed with flexibility in mind - you can add your own executor which will handle additional script types.

## Script test source

Scripts can be currently created from two sources:

- First one is simple `file` with test content e.g. for postman collections we're exporting collection as json file, or for curl executor we're passing json file with configured curl command.
- Second source handled by TestKube is `git` - we can pass `repository`, `path` and `branch` where our tests is. This one is used in Cypress executor - as Cypress tests are more like npm-based projects which can have a lot of files. We're handling here sparse checkouts which are fast even in case of huge mono-repos

## Create script

### Create your first script from file (Postman Collection test)

To create your first postman collection in TestKube you'll first need to export your collection into a file

Right click on your collection name
![create postman colletion step 1](img/script-create-1.png)
Click "Export" button
![create postman colletion step 2](img/script-create-1.png)
Save in convinient location (We're using `~/Downloads/TODO.postman_collection.json` path)
![create postman colletion step 3](img/script-create-1.png)

```sh
kubectl testkube scripts create --file ~/Downloads/TODO.postman_collection.json --name test
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Detected test script type postman/collection
Script created test 🥇
```

Script created! Now we can run as many times as we want.

### Updating scripts

If you need to update your script after change in Postman just re-export it to file and run update command:

```sh
kubectl testkube scripts update --file ~/Downloads/TODO.postman_collection.json --name test
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Detected test script type postman/collection
Script updated test 🥇
```

Testkube will override all script settings and content with `update` method.

### Checking scripts content

Let's see what has been created:

```sh
kubectl get scripts -n testkube
```

Output:

```sh
NAME   AGE
test   32s
```

```sh
$ kubectl testkube scripts get test

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
                                                "type": "text/javascript"
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

We can see that script resource was created with Postman collection JSON content.

You can also check scripts with standard `kubectl` comman which will list Scripts Custom Resource

```sh
kubectl get scripts -ntestkube test -oyaml
```

### Create script from git

Some executors can handle files and some could handle only git resources - You'll need to follow particular executor readme file to be aware what scripts type does he handle.

Let's assume that some Cypress project is created in some git repository - Let's assume we've created one in Let's assume we've created one in <https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples>  

Where `examples` is test directory in `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress based script (in git based scripts we need to pass script type)

```sh
kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Script created kubeshop-cypress 🥇
```

Let's check how created testkube test script is defined on cluster: 

```sh
$ kubectl get scripts -n testkube kubeshop-cypress -o yaml
apiVersion: tests.testkube.io/v1
kind: Script
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

As we can see this script has `spec.repository` with git repository data. Those data can now be used by executor to download script data.

## Summary

Scripts are main smallest abstractions over tests in TestKube, they can be created with different sources and used by executors to run on top of particular test framework.
