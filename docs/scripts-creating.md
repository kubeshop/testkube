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

Create script json file:
```bash
cat <<EOF > my_postman_collection.json
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

```sh
$ kubectl testkube scripts create --name test --file my_postman_collection.json --type postman/collection 

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Script created test 🥇
```

Script created! Now we can run as many times as we want. 

Let's see what has been created:

```sh
$ kubectl get scripts -n testkube
NAME   AGE
test   32s
```

```sh
$ kubectl get scripts -n testkube test -o yaml
apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  creationTimestamp: "2021-11-17T12:26:32Z"
  generation: 1
  name: test
  namespace: testkube
  resourceVersion: "224612"
  uid: c622ef0b-f279-4804-ba76-98c5856fc375
spec:
  content: |
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
  type: postman/collection
```

As we see we get `Script` resource with `spec.content` field which has escaped file content from exported postman collection. and `type` which need to be one handled by executor. `metadata.name` is name passed in testkube scripts create command.


### Create script from git

Some executors can handle files and some could handle only git resources - You'll need to follow particular executor readme file to be aware what scripts type does he handle. 


Let's assume that some Cypress project is created in some git repository - Let's assume we've created one in Let's assume we've created one in https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples  

Where `examples` is test directory in `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress based script

```sh
$ kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project

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


